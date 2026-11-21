<script lang="ts">
	import { resolve } from '$app/paths';

	const contact = 'contact@corcheria.company';
</script>

<svelte:head>
	<title>Política de privacidad · Corcheria Company</title>
	<meta
		name="description"
		content="Qué datos personales recopila Corcheria Company, cómo los usa, con quién los comparte y cómo eliminarlos."
	/>
</svelte:head>

<article
	class="flex flex-col gap-4 [&_h2]:mt-4 [&_h2]:text-xl [&_h2]:font-semibold [&_ul]:list-disc [&_ul]:space-y-1 [&_ul]:pl-6"
>
	<h1 class="text-3xl font-semibold">Política de privacidad</h1>
	<p class="text-sm opacity-70">Última actualización: 21 de noviembre de 2026</p>

	<p>
		Esta política explica qué datos personales recopila Corcheria Company ("nosotros") cuando usás
		el tablero de Corcheria Company en
		<a href={resolve('/about')} class="underline hover:text-main">corcheria.company</a>, por qué los
		recopilamos, con quién los compartimos y qué opciones tenés. Corcheria Company es un tablero
		online compartido donde fijás tarjetas que muestran información en vivo de servicios externos.
	</p>

	<h2>1. Datos que recopilamos</h2>
	<ul>
		<li>
			<strong>Datos de la cuenta.</strong> Tu dirección de correo, tu nombre y, si ingresás con Google,
			tu foto de perfil. Si te registrás con una contraseña, guardamos solo un hash irreversible de ella
			(argon2id), nunca la contraseña. También guardamos tu idioma preferido y si tu dirección de correo
			fue verificada.
		</li>
		<li>
			<strong>Contenido que creás.</strong> Tus tableros, las tarjetas que contienen y su configuración,
			las personas con quienes compartís un tablero y sus roles, y las organizaciones y grupos que creás
			o a los que te sumás.
		</li>
		<li>
			<strong>Credenciales que guardás.</strong> Claves de API y los tokens de acceso y de renovación
			de las cuentas que conectás a un tablero (por ejemplo, Google). Se guardan cifradas con AES-256-GCM
			y nunca se le muestran a nadie, ni siquiera a vos.
		</li>
		<li>
			<strong>Datos de cuentas conectadas.</strong> Cuando una tarjeta usa una cuenta conectada, obtenemos
			los datos que esa tarjeta muestra (ver la sección 3) y guardamos una copia en una caché temporal
			durante el intervalo de actualización de la tarjeta, normalmente unos minutos.
		</li>
		<li>
			<strong>Datos técnicos.</strong> Tu dirección IP, que usamos para limitar los intentos repetidos
			de ingreso, registro y recuperación de contraseña y que guardamos solo mientras dura ese límite,
			y los registros de pedidos del servidor, que conservamos para resolver problemas y por seguridad.
		</li>
		<li>
			<strong>Presencia.</strong> Mientras tenés un tablero abierto registramos que estás conectado a
			él (hasta 20 minutos) y compartimos la posición de tu cursor, tu nombre y tu foto con las otras
			personas que están viendo el mismo tablero.
		</li>
	</ul>

	<h2>2. Cómo los usamos</h2>
	<ul>
		<li>Para crear tu cuenta, hacerte ingresar y mantener tu sesión.</li>
		<li>
			Para mostrar tus tableros y sus tarjetas, y actualizarlos en vivo para las personas con
			quienes están compartidos.
		</li>
		<li>
			Para enviarte correos transaccionales sobre tu cuenta, como verificar tu dirección o
			restablecer tu contraseña. No enviamos correos de marketing.
		</li>
		<li>Para proteger el servicio contra abusos y para resolver problemas.</li>
	</ul>
	<p>
		No vendemos tus datos, no mostramos publicidad y no usamos tus datos para armar perfiles ni para
		entrenar modelos de aprendizaje automático.
	</p>

	<h2>3. Datos de usuario de Google</h2>
	<p>
		<strong>Ingresar con Google.</strong> Cuando ingresás con Google pedimos los permisos
		<code>openid</code>, <code>email</code> y <code>profile</code> y recibimos tu nombre, tu dirección
		de correo y tu foto de perfil. Los usamos solo para crear e identificar tu cuenta.
	</p>
	<p>
		<strong>Tarjetas de Gmail y Google Calendar.</strong> Opcionalmente, podés conectar tu cuenta de
		Google a un tablero. En ese caso pedimos acceso de solo lectura a Gmail (<code
			>https://www.googleapis.com/auth/gmail.readonly</code
		>) y a Google Calendar (<code>https://www.googleapis.com/auth/calendar.readonly</code>):
	</p>
	<ul>
		<li>
			La tarjeta <strong>Gmail Inbox</strong> lee la cantidad de mensajes sin leer y la cantidad total
			de mensajes de tu bandeja de entrada y muestra la cantidad sin leer en el tablero. No lee el contenido,
			los remitentes ni los asuntos de tus mensajes.
		</li>
		<li>
			La tarjeta <strong>Google Calendar</strong> lee los próximos eventos de tu calendario principal
			y muestra en el tablero el título y la hora de inicio del siguiente.
		</li>
	</ul>
	<p>
		Estos datos se obtienen solo cuando se muestra la tarjeta, se ven solo en el tablero donde
		agregaste la tarjeta (para vos y para las personas con quienes elegiste compartir ese tablero) y
		se guardan solo en una caché temporal durante el intervalo de actualización de la tarjeta. Nunca
		enviamos, modificamos ni eliminamos tus correos o eventos, nunca guardamos tus correos o eventos
		de forma permanente y nunca usamos datos de usuario de Google para publicidad, para entrenar
		modelos de inteligencia artificial o de aprendizaje automático, ni para ningún otro fin que no
		sea mostrarlos en tu tarjeta. Ninguna persona de Corcheria Company lee tus datos de Google,
		salvo con tu permiso explícito para investigar un problema que nos reportes, cuando sea
		necesario por seguridad o cuando lo exija la ley.
	</p>
	<p>
		El uso y la transferencia que hace Corcheria Company de la información recibida de las APIs de
		Google cumplen con la
		<a
			href="https://developers.google.com/terms/api-services-user-data-policy"
			class="underline hover:text-main"
			rel="noopener noreferrer"
			>Política de datos del usuario de los servicios de las APIs de Google</a
		>, incluidos los requisitos de uso limitado.
	</p>
	<p>
		Podés desconectar tu cuenta de Google en cualquier momento eliminando la credencial desde el
		panel de credenciales del tablero, o quitándole el acceso a Corcheria Company en
		<a
			href="https://myaccount.google.com/permissions"
			class="underline hover:text-main"
			rel="noopener noreferrer">myaccount.google.com/permissions</a
		>. Eliminar la credencial borra los tokens guardados.
	</p>

	<h2>4. Con quién compartimos los datos</h2>
	<p>
		Compartimos datos solo con los proveedores de servicios que operan Corcheria Company por
		nosotros, y solo en la medida necesaria para eso:
	</p>
	<ul>
		<li>
			Nuestros proveedores de alojamiento, base de datos y caché, que guardan y procesan los datos
			descriptos arriba.
		</li>
		<li>
			Vercel Speed Insights, que mide qué tan rápido cargan y responden nuestras páginas en tu
			navegador. Recibe esos tiempos, la dirección de la página sin su query string y el tipo de
			dispositivo y de navegador. No usa cookies y no te identifica.
		</li>
		<li>
			Resend, que entrega nuestros correos transaccionales (tu dirección de correo y el contenido
			del correo).
		</li>
		<li>Google, cuando ingresás con Google o conectás una cuenta de Google.</li>
		<li>
			El servidor de señalización de PeerJS, que conecta tu navegador directamente con los de las
			otras personas que están viendo el mismo tablero para que se puedan compartir los cursores.
			Los navegadores de esas personas pueden ver tu dirección IP.
		</li>
		<li>
			Los servicios externos que consultan tus tarjetas (por ejemplo, APIs de clima o de
			cotizaciones). Reciben el pedido que hace la tarjeta, desde nuestros servidores, y la
			credencial que configuraste para ella, si la hay.
		</li>
	</ul>
	<p>
		Las personas con quienes compartís un tablero ven sus tarjetas, incluidos los resultados de las
		tarjetas que usan tus cuentas conectadas. También podemos revelar datos si lo exige la ley.
	</p>

	<h2>5. Cookies y almacenamiento local</h2>
	<ul>
		<li>
			<code>tiu_refresh</code>: una cookie segura y http-only que mantiene tu sesión iniciada hasta
			30 días.
		</li>
		<li>Una cookie que recuerda tu idioma preferido.</li>
		<li>Almacenamiento del navegador que recuerda la última organización que elegiste.</li>
	</ul>
	<p>
		No usamos cookies de publicidad ni de seguimiento de terceros. El rendimiento de las páginas se
		mide con Vercel Speed Insights (ver la sección 4), que no usa cookies.
	</p>

	<h2>6. Conservación y eliminación</h2>
	<p>
		Conservamos los datos de tu cuenta y tu contenido mientras tu cuenta exista. Eliminar un
		tablero, salir de él o ser quitado de él borra las credenciales que se guardaban en ese tablero
		para vos. Podés pedirnos que eliminemos tu cuenta y todos los datos asociados en cualquier
		momento escribiendo a
		<a href="mailto:{contact}" class="underline hover:text-main">{contact}</a>; lo haremos dentro de
		los 30 días, salvo los datos que debamos conservar por ley.
	</p>

	<h2>7. Seguridad</h2>
	<p>
		Todo el tráfico se cifra con HTTPS. Las contraseñas se guardan como hash, las credenciales y los
		tokens se cifran en reposo y el acceso a cada tablero se verifica en cada pedido. Ningún sistema
		es perfectamente seguro, pero trabajamos para proteger tus datos y te avisaremos de cualquier
		filtración que los afecte, según lo exija la ley.
	</p>

	<h2>8. Tus derechos</h2>
	<p>
		Podés ver y corregir tu perfil desde tu cuenta, y pedirnos una copia de tus datos, su corrección
		o su eliminación escribiendo a
		<a href="mailto:{contact}" class="underline hover:text-main">{contact}</a>. Corcheria Company no
		está dirigida a menores de 13 años y no recopilamos sus datos a sabiendas.
	</p>

	<h2>9. Cambios en esta política</h2>
	<p>
		Si cambiamos esta política actualizaremos la fecha de arriba y, ante cambios importantes, te
		avisaremos por correo o en la aplicación.
	</p>

	<h2>10. Contacto</h2>
	<p>
		Consultas sobre esta política o sobre tus datos:
		<a href="mailto:{contact}" class="underline hover:text-main">{contact}</a>.
	</p>
</article>
