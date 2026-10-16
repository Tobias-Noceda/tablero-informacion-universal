import { goto } from '$app/navigation';
import { resolve } from '$app/paths';
import * as auth from '$services/auth';
import type { Profile, SessionResponse, UUID } from '$types/api';
import * as api from './api.svelte';

export type SessionStatus = 'unknown' | 'anonymous' | 'authenticated';

export type UserChangeListener = (userId: UUID | null) => void;

type TabMessage = { type: 'signed-in'; session: SessionResponse } | { type: 'signed-out' };

// The signed-in user and their access token. The token lives only in memory;
// the refresh token is an httpOnly cookie the browser sends to /api/v1/auth,
// so a reload or a new tab gets back in with one POST /auth/refresh.
export class Session implements api.Authenticator {
	user = $state<Profile | null>(null);
	status = $state<SessionStatus>('unknown');

	#token: string | null = null;
	#expiresAt = 0;
	#booting: Promise<void> | null = null;
	#refreshing: Promise<boolean> | null = null;
	#channel: BroadcastChannel | null = null;
	#listeners = new Set<UserChangeListener>();

	constructor(private readonly channelName = 'tiu-session') {}

	// The signed-in user's id. Only read it where a user is guaranteed: the
	// (app) routes and what they call.
	get userId() {
		if (!this.user) throw new Error('No signed-in user');
		return this.user.id;
	}

	// When the access token stops working, in epoch milliseconds.
	get expiresAt() {
		return this.#expiresAt;
	}

	get accessToken() {
		return this.#token;
	}

	header() {
		return this.#token && `Bearer ${this.#token}`;
	}

	bootstrap() {
		this.#booting ??= this.#listen()
			.refresh()
			.then(() => undefined);
		return this.#booting;
	}

	// Single-flight: every caller waiting on a renewal shares the same request,
	// since the cookie rotates and a second one would arrive already spent.
	refresh() {
		this.#refreshing ??= this.#withLock(() => this.#rotate()).finally(() => {
			this.#refreshing = null;
		});
		return this.#refreshing;
	}

	renew() {
		return this.refresh();
	}

	// Called whenever the signed-in user becomes someone else (or nobody), so
	// stores keeping what the previous one could see start over. A renewal
	// that keeps the same user does not call it.
	onUserChange(listener: UserChangeListener) {
		this.#listeners.add(listener);
		return () => {
			this.#listeners.delete(listener);
		};
	}

	expired() {
		this.#clear();
		const here = globalThis.location;
		if (!here || here.pathname === '/login') return;
		void goto(resolve(`/login?next=${encodeURIComponent(here.pathname + here.search)}`));
	}

	async login(email: string, password: string) {
		this.#signIn(await auth.login(email, password));
	}

	register(email: string, password: string, name: string) {
		return auth.register(email, password, name);
	}

	async verify(token: string) {
		this.#signIn(await auth.verifyEmail(token));
	}

	forgot(email: string) {
		return auth.forgotPassword(email);
	}

	async reset(token: string, password: string) {
		this.#signIn(await auth.resetPassword(token, password));
	}

	googleStart(next: string) {
		return auth.googleStart(next);
	}

	async googleCallback(code: string, state: string) {
		const { next, ...session } = await auth.googleCallback(code, state);
		this.#signIn(session);
		return next;
	}

	async logout() {
		try {
			await auth.logout();
		} catch (err) {
			console.error('Logout did not reach the server', err);
		}
		this.#clear();
		this.#broadcast({ type: 'signed-out' });
	}

	dispose() {
		this.#channel?.close();
		this.#channel = null;
	}

	async #rotate() {
		const stale = this.#token;
		try {
			this.#adopt(await auth.refresh());
			return true;
		} catch (err) {
			// Another tab may have rotated the cookie and handed us its token.
			if (this.#token && this.#token !== stale) return true;
			if (!(err instanceof auth.AuthError)) console.error('Session refresh failed', err);
			this.#clear();
			return false;
		}
	}

	// Tabs share the cookie: without a lock two of them could present the same
	// refresh token at once. Plain-HTTP origins have no Web Locks and rely on
	// the backend's reuse grace instead.
	#withLock<T>(task: () => Promise<T>): Promise<T> {
		const locks = globalThis.navigator?.locks;
		return locks ? locks.request('tiu-refresh', task) : task();
	}

	#signIn(session: SessionResponse) {
		this.#adopt(session);
		this.#broadcast({ type: 'signed-in', session });
	}

	#adopt(session: SessionResponse) {
		this.#switchTo(session.user.id);
		this.#token = session.access_token;
		this.#expiresAt = Date.now() + session.expires_in * 1000;
		this.user = session.user;
		this.status = 'authenticated';
	}

	#clear() {
		this.#switchTo(null);
		this.#token = null;
		this.#expiresAt = 0;
		this.user = null;
		this.status = 'anonymous';
	}

	// Runs before the new user is visible, so nothing reacting to the status
	// reads the previous one's data.
	#switchTo(userId: UUID | null) {
		if ((this.user?.id ?? null) === userId) return;
		for (const listener of this.#listeners) {
			try {
				listener(userId);
			} catch (err) {
				console.error('User change listener failed', err);
			}
		}
	}

	#listen() {
		if (!this.#channel && typeof BroadcastChannel !== 'undefined') {
			this.#channel = new BroadcastChannel(this.channelName);
			this.#channel.onmessage = ({ data }: MessageEvent<TabMessage>) => {
				if (data.type === 'signed-in') this.#adopt(data.session);
				else this.#clear();
			};
		}
		return this;
	}

	#broadcast(message: TabMessage) {
		this.#listen().#channel?.postMessage(message);
	}
}

export const session = new Session();
api.authenticate(session);
