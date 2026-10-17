<script lang="ts">
	import { resolve } from '$app/paths';

	const contact = 'contact@corcheria.company';
</script>

<svelte:head>
	<title>Privacy Policy · Corcheria Company</title>
	<meta
		name="description"
		content="What personal data Corcheria Company collects, how it is used, who it is shared with and how to delete it."
	/>
</svelte:head>

<article
	class="flex flex-col gap-4 [&_h2]:mt-4 [&_h2]:text-xl [&_h2]:font-semibold [&_ul]:list-disc [&_ul]:space-y-1 [&_ul]:pl-6"
>
	<h1 class="text-3xl font-semibold">Privacy Policy</h1>
	<p class="text-sm opacity-70">Last updated: October 16, 2026</p>

	<p>
		This policy explains what personal data Corcheria Company ("we", "us") collects when you use the
		Corcheria Company board at
		<a href={resolve('/about')} class="underline hover:text-main">corcheria.company</a>, why we
		collect it, who we share it with and the choices you have. Corcheria Company is a shared online
		board where you pin cards that show live information from external services.
	</p>

	<h2>1. Data we collect</h2>
	<ul>
		<li>
			<strong>Account data.</strong> Your email address, name and, if you sign in with Google, your profile
			picture. If you register with a password we store only a one-way hash of it (argon2id), never the
			password itself. We also store your preferred language and whether your email address has been verified.
		</li>
		<li>
			<strong>Content you create.</strong> Your boards, the cards on them and their settings, the people
			you share a board with and their roles, and the organizations and groups you create or join.
		</li>
		<li>
			<strong>Credentials you store.</strong> API keys and the access and refresh tokens of accounts you
			connect to a board (for example Google). They are encrypted at rest with AES-256-GCM and are never
			shown back to anyone, including you.
		</li>
		<li>
			<strong>Data from connected accounts.</strong> When a card uses a connected account, we fetch the
			data that card displays (see section 3) and keep a copy in a temporary cache for the card's refresh
			interval, typically a few minutes.
		</li>
		<li>
			<strong>Technical data.</strong> Your IP address, used to limit repeated sign-in, registration and
			password reset attempts and kept only for the length of that limit, and server logs of requests,
			kept for troubleshooting and security.
		</li>
		<li>
			<strong>Presence.</strong> While a board is open we record that you are online on it (for up to
			20 minutes) and share your mouse cursor position, name and picture with the other people viewing
			the same board.
		</li>
	</ul>

	<h2>2. How we use it</h2>
	<ul>
		<li>To create your account, sign you in and keep you signed in.</li>
		<li>
			To show your boards and their cards, and to update them live for the people they are shared
			with.
		</li>
		<li>
			To send you transactional emails about your account, such as verifying your address or
			resetting your password. We do not send marketing emails.
		</li>
		<li>To protect the service against abuse and to fix problems.</li>
	</ul>
	<p>
		We do not sell your data, we do not show advertising and we do not use your data to build
		profiles or to train machine learning models.
	</p>

	<h2>3. Google user data</h2>
	<p>
		<strong>Sign in with Google.</strong> When you sign in with Google we request the
		<code>openid</code>, <code>email</code> and <code>profile</code> scopes and receive your name, email
		address and profile picture. We use them only to create and identify your account.
	</p>
	<p>
		<strong>Gmail and Google Calendar cards.</strong> You can optionally connect your Google account
		to a board. We then request read-only access to Gmail (<code
			>https://www.googleapis.com/auth/gmail.readonly</code
		>) and Google Calendar (<code>https://www.googleapis.com/auth/calendar.readonly</code>):
	</p>
	<ul>
		<li>
			The <strong>Gmail Inbox</strong> card reads the number of unread and total messages in your inbox
			and shows the unread count on the board. It does not read the content, senders or subjects of your
			messages.
		</li>
		<li>
			The <strong>Google Calendar</strong> card reads the upcoming events of your primary calendar and
			shows the title and start time of the next one on the board.
		</li>
	</ul>
	<p>
		This data is fetched only when the card is displayed, is shown only on the board where you added
		the card (to you and to the people you chose to share that board with), and is kept only in a
		temporary cache for the card's refresh interval. We never send, modify or delete your emails or
		events, never store your emails or events permanently, and never use Google user data for
		advertising, for training AI or machine learning models, or for any purpose other than
		displaying it on your card. No person at Corcheria Company reads your Google data, except with
		your explicit permission to investigate a problem you report, when required for security or by
		law.
	</p>
	<p>
		Corcheria Company's use and transfer of information received from Google APIs adheres to the
		<a
			href="https://developers.google.com/terms/api-services-user-data-policy"
			class="underline hover:text-main"
			rel="noopener noreferrer">Google API Services User Data Policy</a
		>, including the Limited Use requirements.
	</p>
	<p>
		You can disconnect your Google account at any time by deleting the credential from the board's
		credentials panel, or by removing Corcheria Company's access at
		<a
			href="https://myaccount.google.com/permissions"
			class="underline hover:text-main"
			rel="noopener noreferrer">myaccount.google.com/permissions</a
		>. Deleting the credential erases the stored tokens.
	</p>

	<h2>4. Who we share data with</h2>
	<p>
		We share data only with the service providers that run Corcheria Company for us, and only as
		needed for that:
	</p>
	<ul>
		<li>Our hosting, database and cache providers, which store and process the data above.</li>
		<li>
			Resend, which delivers our transactional emails (your email address and the email's content).
		</li>
		<li>Google, when you sign in with Google or connect a Google account.</li>
		<li>
			The PeerJS signaling server, which connects your browser directly to the other people viewing
			the same board so that cursors can be shared. Those people's browsers can see your IP address.
		</li>
		<li>
			The external services your cards query (for example weather or exchange rate APIs). They
			receive the request the card makes, from our servers, and the credential you configured for
			it, if any.
		</li>
	</ul>
	<p>
		People you share a board with see its cards, including the results of cards that use your
		connected accounts. We may also disclose data if required by law.
	</p>

	<h2>5. Cookies and local storage</h2>
	<ul>
		<li>
			<code>tiu_refresh</code>: a secure, http-only cookie that keeps you signed in for up to 30
			days.
		</li>
		<li>A cookie that remembers your preferred language.</li>
		<li>Browser storage that remembers the last organization you selected.</li>
	</ul>
	<p>We do not use analytics, advertising or third-party tracking cookies.</p>

	<h2>6. Retention and deletion</h2>
	<p>
		We keep your account data and content while your account exists. Deleting a board, leaving it or
		being removed from it erases the credentials kept on that board for you. You can ask us to
		delete your account and all associated data at any time by writing to
		<a href="mailto:{contact}" class="underline hover:text-main">{contact}</a>; we will do so within
		30 days, except for data we must keep by law.
	</p>

	<h2>7. Security</h2>
	<p>
		All traffic is encrypted with HTTPS. Passwords are hashed, credentials and tokens are encrypted
		at rest, and access to each board is checked on every request. No system is perfectly secure,
		but we work to protect your data and will notify you of any breach that affects it as required
		by law.
	</p>

	<h2>8. Your rights</h2>
	<p>
		You can access and correct your profile from your account, and ask us for a copy of your data,
		its correction or its deletion by writing to
		<a href="mailto:{contact}" class="underline hover:text-main">{contact}</a>. Corcheria Company is
		not directed at children under 13 and we do not knowingly collect their data.
	</p>

	<h2>9. Changes to this policy</h2>
	<p>
		If we change this policy we will update the date above, and for significant changes we will
		notify you by email or in the app.
	</p>

	<h2>10. Contact</h2>
	<p>
		Questions about this policy or your data:
		<a href="mailto:{contact}" class="underline hover:text-main">{contact}</a>.
	</p>
</article>
