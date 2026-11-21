<script lang="ts">
	import { resolve } from '$app/paths';
</script>

<svelte:head>
	<title>Corcheria Company: un tablero en vivo con la información que revisás todos los días</title>
	<meta
		name="description"
		content="Corcheria Company es un tablero online compartido donde fijás tarjetas que muestran datos en vivo de los servicios que usás: el clima, cotizaciones, tus correos sin leer de Gmail, tu próximo evento de Google Calendar y más."
	/>
</svelte:head>

<section class="flex flex-col gap-4">
	<h1 class="text-3xl font-semibold">Corcheria Company</h1>
	<p class="text-lg">
		Corcheria Company es un tablero online compartido, como un pizarrón de corcho, donde fijás
		tarjetas que muestran información en vivo de los servicios que ya usás. Cada tarjeta trae datos
		de una fuente (el clima, una cotización, un repositorio de GitHub, tus correos sin leer, tu
		próxima reunión) y los mantiene actualizados, así todo lo que revisás durante el día está en una
		sola pantalla.
	</p>
	<div class="flex flex-row flex-wrap gap-3">
		<a
			href={resolve('/register')}
			class="rounded-md bg-main px-4 py-2 font-medium text-main-text hover:bg-main-hover"
		>
			Crear una cuenta
		</a>
		<a href={resolve('/login')} class="rounded-md border border-border px-4 py-2 font-medium">
			Ingresar
		</a>
		<a href={resolve('/pricing')} class="rounded-md border border-border px-4 py-2 font-medium">
			Planes y precios
		</a>
	</div>
</section>

<section class="flex flex-col gap-2">
	<h2 class="text-xl font-semibold">Qué podés hacer</h2>
	<ul class="list-disc space-y-1 pl-6">
		<li>Crear tableros y acomodar tarjetas libremente en un lienzo.</li>
		<li>
			Agregar tarjetas listas para usar (temperatura, calidad del aire, cotizaciones, precios de
			criptomonedas, la foto del día de la NASA, repositorios de GitHub) o configurar las tuyas
			desde cualquier API web.
		</li>
		<li>
			Compartir un tablero con otras personas como editoras o lectoras; los cambios aparecen en vivo
			para todos los que lo tienen abierto.
		</li>
		<li>
			Guardar las claves de API y las cuentas conectadas que necesitan tus tarjetas en una bóveda
			cifrada, para todo el tablero o solo para vos.
		</li>
	</ul>
</section>

<section class="flex flex-col gap-2">
	<h2 class="text-xl font-semibold">Cómo usa Corcheria Company tu cuenta de Google</h2>
	<p>
		Podés ingresar con Google. En ese caso Corcheria Company recibe tu nombre, tu dirección de
		correo y tu foto de perfil para crear e identificar tu cuenta.
	</p>
	<p>
		Si querés, también podés conectar tu cuenta de Google para mostrar tus propios datos en un
		tablero: la tarjeta <strong>Gmail Inbox</strong> muestra cuántos correos sin leer tenés y la
		tarjeta <strong>Google Calendar</strong> muestra tu próximo evento. Estas tarjetas solo leen datos
		(acceso de solo lectura); Corcheria Company nunca envía, modifica ni elimina tus correos o eventos.
		Los datos se muestran en el tablero donde agregás la tarjeta y no se usan para nada más. Podés desconectar
		tu cuenta en cualquier momento.
	</p>
	<p>
		Leé la <a href={resolve('/privacy')} class="underline hover:text-main">Política de privacidad</a
		>
		para conocer en detalle qué datos recopilamos y cómo los usamos.
	</p>
</section>
