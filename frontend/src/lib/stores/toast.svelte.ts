type Variant = 'normal' | 'destructive' | 'success';

// The one notice the (app) layout shows through `Toast`; a new one replaces it.
class Notice {
	show = $state(false);
	title = $state('');
	description = $state('');
	variant = $state<Variant>('normal');

	notify(notice: { title?: string; description?: string; variant?: Variant }) {
		this.title = notice.title ?? '';
		this.description = notice.description ?? '';
		this.variant = notice.variant ?? 'normal';
		this.show = true;
	}
}

export const toast = new Notice();
