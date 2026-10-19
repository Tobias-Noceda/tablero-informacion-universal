<script lang="ts">
	import { resolve } from '$app/paths';
</script>

<svelte:head>
	<title>Corcheria Company: a live board for the information you check every day</title>
	<meta
		name="description"
		content="Corcheria Company is a shared online board where you pin cards that show live data from the services you use: weather, exchange rates, your Gmail unread count, your next Google Calendar event and more."
	/>
</svelte:head>

<section class="flex flex-col gap-4">
	<h1 class="text-3xl font-semibold">Corcheria Company</h1>
	<p class="text-lg">
		Corcheria Company is a shared online board, like a cork board, where you pin cards that show
		live information from the services you already use. Each card fetches data from one source (the
		weather, an exchange rate, a GitHub repository, your unread emails, your next meeting) and keeps
		it up to date, so everything you check during the day is on one screen.
	</p>
	<div class="flex flex-row flex-wrap gap-3">
		<a
			href={resolve('/register')}
			class="rounded-md bg-main px-4 py-2 font-medium text-main-text hover:bg-main-hover"
		>
			Create an account
		</a>
		<a href={resolve('/login')} class="rounded-md border border-border px-4 py-2 font-medium">
			Sign in
		</a>
	</div>
</section>

<section class="flex flex-col gap-2">
	<h2 class="text-xl font-semibold">What you can do</h2>
	<ul class="list-disc space-y-1 pl-6">
		<li>Create boards and arrange cards freely on a canvas.</li>
		<li>
			Add ready-made cards (temperature, air quality, exchange rates, crypto prices, NASA's picture
			of the day, GitHub repositories) or configure your own from any web API.
		</li>
		<li>
			Share a board with other people as editors or viewers; changes appear live for everyone who
			has it open.
		</li>
		<li>
			Keep the API keys and connected accounts your cards need in an encrypted vault, for the whole
			board or only for yourself.
		</li>
	</ul>
</section>

<section class="flex flex-col gap-2">
	<h2 class="text-xl font-semibold">How Corcheria Company uses your Google account</h2>
	<p>
		You can sign in with Google. Corcheria Company then receives your name, email address and
		profile picture to create and identify your account.
	</p>
	<p>
		Optionally, you can connect your Google account to show your own data on a board: the
		<strong>Gmail Inbox</strong> card shows how many unread emails you have and the
		<strong>Google Calendar</strong> card shows your next event. These cards only read data (read-only
		access); Corcheria Company never sends, changes or deletes your emails or events. The data is shown
		on the board where you add the card and is not used for anything else. You can disconnect your account
		at any time.
	</p>
	<p>
		Read the <a href={resolve('/privacy')} class="underline hover:text-main">Privacy Policy</a> for the
		full details of what we collect and how we use it.
	</p>
</section>
