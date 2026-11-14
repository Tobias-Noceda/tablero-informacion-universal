# Plan de monetización y escalamiento de recursos

Tablero de información universal. Tarifas consultadas en las páginas oficiales de cada proveedor el 13 de noviembre de 2026. Todos los importes están en dólares estadounidenses por mes, sin impuestos.

## 1. Resumen ejecutivo

Se proponen tres planes: Free, Pro a USD 9 mensuales y Max a USD 29 mensuales, cobrados en pesos por Mercado Pago.

Con una conversión a pago del 3 % de los usuarios, el ingreso neto supera al costo de infraestructura en todos los hitos analizados. La infraestructura representa el 13 % del ingreso neto con 1.000 usuarios y menos del 3 % con 5 millones.

La arquitectura actual (Vercel, MongoDB Atlas, Redis Cloud y Resend) es la opción más económica hasta unos 500.000 usuarios. La migración del cómputo a AWS se justifica a partir de 1.000.000 de usuarios, que equivalen a 100.000 concurrentes. MongoDB Atlas se conserva en todos los hitos.

## 2. Uso de los servicios y supuestos de consumo

### 2.1 Uso verificado en el código

- **Vercel.** Aloja cuatro servicios bajo un mismo origen: la aplicación web, la API en Go, el servicio de tiempo real en Node y el servidor MCP. Cada solicitud a la API es una invocación de función. Cada pestaña abierta mantiene un WebSocket con el servicio de tiempo real.
- **MongoDB Atlas.** Base principal. Requiere un conjunto de réplicas porque el servicio de tiempo real escucha los cambios de la colección de tableros. Cada consulta de una tarjeta realiza dos lecturas (la tarjeta y su tablero) aunque el resultado esté en caché.
- **Redis Cloud.** Guarda el resultado de cada tarjeta durante su intervalo de actualización, las sesiones, los límites de tasa y la presencia en línea. Cada consulta de una tarjeta realiza una lectura.
- **Resend.** Envía tres correos: verificación de cuenta, aviso de cuenta existente y restablecimiento de contraseña.

### 2.2 Supuestos

1. El pico de usuarios concurrentes es el 10 % de los usuarios totales. El objetivo de 500.000 concurrentes equivale a 5.000.000 de usuarios.
2. La concurrencia promedio del mes es el 30 % del pico.
3. Un usuario concurrente mantiene abierto un tablero con 10 tarjetas que consultan una fuente externa. La aplicación consulta cada tarjeta con un intervalo medio de 300 segundos, que es el mínimo del plan Free.
4. De lo anterior resultan 0,038 solicitudes por segundo por usuario concurrente, incluida la carga del tablero y la reconexión del WebSocket, y 3.200 solicitudes mensuales por usuario total. La red de distribución suma 800 solicitudes de archivos estáticos por usuario.
5. Cada solicitud consume 5 milisegundos de CPU y transfiere 1,5 KB.
6. Los correos mensuales equivalen al 15 % de los usuarios totales.
7. Todas las regiones son de Estados Unidos (Vercel iad1, AWS us-east-1, Atlas y Redis Cloud en us-east-1). No se busca cercanía geográfica. São Paulo se descarta porque encarece el cómputo un 72 % y la salida de datos un 67 %.

## 3. Costo por hito de crecimiento

Tarifas de Vercel aplicadas: USD 20 de plan Pro con USD 20 de crédito, USD 0,60 por millón de invocaciones, USD 0,128 por hora de CPU activa, USD 0,0106 por GB hora de memoria, USD 0,06 por GB de transferencia de origen y USD 2 por millón de solicitudes de red por encima del nivel plano contratado.

| Usuarios (pico concurrente) | Infraestructura                                              | Costo mensual |
| --------------------------- | ------------------------------------------------------------ | ------------- |
| 1.000 (100)                 | Vercel 34, Atlas Flex 8, Redis 250 MB 5, Resend gratuito     | 47            |
| 10.000 (1.000)              | Vercel 160, Atlas M10 57, Redis 1 GB 18, Resend Pro 20       | 255           |
| 100.000 (10.000)            | Vercel 1.150, Atlas M30 394, Redis 2,5 GB 52, Resend Pro 20  | 1.620         |
| 500.000 (50.000)            | Vercel 5.700, Atlas M50 1.460, Redis 5 GB 100, Resend Pro 35 | 7.300         |
| 1.000.000 (100.000)         | AWS 1.700, Atlas M60 2.884, Redis 5 GB 100, SES 15           | 4.700         |
| 5.000.000 (500.000)         | AWS 7.800, Atlas fragmentado 8.650, ElastiCache 770, SES 75  | 17.300        |

A partir de 1.000.000 de usuarios se suma el costo de operar infraestructura propia, tratado en la sección 4.

### 3.1 Saltos de costo por proveedor

**Vercel.**

- El plan Hobby solo admite uso no comercial. Desde el primer cobro se requiere el plan Pro.
- Las solicitudes de red son el componente dominante. El nivel plano más alto cubre 150 millones de solicitudes por USD 300. Ese volumen se alcanza cerca de 40.000 usuarios. Por encima se factura por uso y llega al 70 % de la factura de Vercel.
- Cada instancia admite 1.024 descriptores de archivo, lo que limita a unos 800 los WebSockets por instancia. Las conexiones se cierran al cumplirse la duración máxima de la función y el cliente debe reconectar. Con 500.000 concurrentes se necesitan más de 600 instancias permanentes.
- El plan Pro admite 30.000 ejecuciones concurrentes. Por encima se requiere el plan Enterprise.

**MongoDB Atlas.**

- El nivel gratuito M0 admite 100 operaciones por segundo y no ofrece respaldos. No es apto para producción.
- Flex cuesta entre USD 8 y 30 y admite hasta 500 operaciones por segundo, suficiente por capacidad hasta unos 50.000 usuarios.
- Los niveles dedicados comienzan en M10 (USD 57). Se adopta desde 10.000 usuarios por sus respaldos y recursos propios. La fragmentación exige M30 como mínimo.
- Con 5.000.000 de usuarios el pico estimado es de 42.000 operaciones por segundo. Guardar en Redis el tablero y la tarjeta elimina las dos lecturas por consulta y permite permanecer en un nivel M60 sin fragmentar.

**Redis Cloud.**

- El nivel gratuito admite 30 conexiones. Cada instancia de función abre su propia conexión, por lo que resulta insuficiente desde el inicio.
- Los planes Essentials llegan a 12 GB y 16.000 operaciones por segundo (USD 234). Por encima corresponde el plan Pro, con un mínimo de USD 200 y tarifas de red no publicadas.

**Resend.**

- El nivel gratuito admite 3.000 correos mensuales con un tope de 100 diarios. Se adopta el plan Pro desde 10.000 usuarios porque un pico de registros supera el tope diario.
- El plan de 1.000.000 de correos cuesta USD 650. El mismo volumen en Amazon SES cuesta USD 100.

## 4. Alternativa en AWS y criterio de migración

### 4.1 Arquitectura equivalente

El proyecto ya dispone de una topología en contenedores con un enrutador nginx por prefijo de ruta. En AWS esa topología se ejecuta sobre ECS Fargate con procesadores ARM detrás de un Application Load Balancer.

Tarifas aplicadas: USD 28,8 por tarea de 1 vCPU y 2 GB, USD 16,4 por balanceador más USD 0,008 por unidad de capacidad y hora, USD 32,9 por NAT Gateway y USD 0,09 por GB de salida de datos.

| Componente                              | Con 1.000.000 de usuarios | Con 5.000.000 de usuarios |
| --------------------------------------- | ------------------------- | ------------------------- |
| API, tiempo real y enrutador en Fargate | 580                       | 2.700                     |
| Balanceador y NAT Gateway               | 280                       | 1.070                     |
| Salida de datos y registros             | 870                       | 4.050                     |
| Total de cómputo y red                 | 1.700                     | 7.800                     |

Se supone una capacidad de 10.000 WebSockets por tarea de tiempo real y de 500 solicitudes por segundo por tarea de API. Ambos valores son conservadores y deben validarse con una prueba de carga.

### 4.2 Servicios que se reemplazan y servicios que se conservan

- **Cómputo.** Se reemplaza Vercel por Fargate. Es el único cambio con ahorro significativo.
- **Base de datos.** Se conserva MongoDB Atlas, que también opera sobre AWS. Amazon DocumentDB se descarta porque sus flujos de cambios están desactivados por defecto, retienen los eventos 3 horas y admiten solo parte de las operaciones que utiliza el servicio de tiempo real.
- **Caché.** Se conserva Redis Cloud hasta superar las 16.000 operaciones por segundo. Luego se adopta ElastiCache con Valkey (USD 128 por nodo r7g.large).
- **Correo.** El ahorro de SES es de USD 145 mensuales con 1.000.000 de usuarios y de USD 575 con 5.000.000. No justifica un cambio aislado. Se realiza junto con la migración del cómputo y requiere reemplazar un único componente de envío.

### 4.3 Criterio

La migración tiene dos costos. El primero es puntual: entre 4 y 6 semanas de ingeniería, estimadas en USD 20.000, para reconstruir el despliegue, los entornos de prueba, los registros y la supervisión que Vercel incluye. El segundo es permanente: la operación de la infraestructura, estimada en USD 4.000 mensuales.

La migración se justifica cuando el ahorro mensual, descontada la operación, recupera el costo puntual en menos de 6 meses.

| Usuarios  | Costo en Vercel y en AWS con operación | Ahorro mensual neto |
| --------- | --------------------------------------- | ------------------- |
| 500.000   | 7.300 y 6.800                           | 500                 |
| 1.000.000 | 14.500 y 8.700                          | 5.800               |
| 5.000.000 | 67.000 y 21.300                         | 45.700              |

Con 500.000 usuarios el ahorro no cubre la migración en un plazo razonable. Con 1.000.000 se recupera en menos de 4 meses. Por lo tanto la migración se planifica al superar los 500.000 usuarios y se ejecuta antes de 1.000.000.

Antes de ese punto corresponde reducir el consumo dentro de Vercel: guardar en Redis las lecturas repetidas y negociar un plan Enterprise si el nivel plano de red resulta insuficiente.

## 5. Planes

El plan pertenece al usuario. Los límites de un tablero dependen de quien lo paga: su dueño si es personal o el dueño de la organización si pertenece a una. Un usuario del plan Free puede participar sin costo en tableros y organizaciones de otros usuarios.

| Límite                              | Free    | Pro y Max    |
| ------------------------------------ | ------- | ------------ |
| Organizaciones propias               | Ninguna | 3 y 10       |
| Tableros por espacio                 | 5       | 10 y 50      |
| Personas por tablero u organización | 1       | 5 y 25       |
| Nodos por tablero                    | 25      | 100 y 300    |
| Plantillas de usuario                | 5       | 10 y 50      |
| Plantillas por tablero               | Ninguna | 5 y 50       |
| Plantillas por organización         | Ninguna | 3 y 50       |
| Credenciales almacenadas             | 5       | 25 y 200     |
| Grupos de credenciales               | Ninguno | 5 y 25       |
| Intervalo mínimo de actualización  | 300 s   | 150 s y 60 s |
| Actualización manual sin caché     | No      | No y Sí     |
| MCP y tokens de acceso               | No      | Sí y Sí    |

Criterios de diseño:

- **Intervalo mínimo.** Es el principal generador de costo y protege a las fuentes externas de un exceso de consultas. Se aplica como piso del tiempo de caché.
- **Colaboración.** Es la diferencia principal entre Free y Pro. En Pro el total de personas con acceso a un tablero, contando a los miembros de la organización y a los invitados externos, no supera 5.
- **Actualización manual.** Queda reservada a Max. La aplicación no guarda en caché las ejecuciones fallidas, por lo que una consulta común ya reintenta.
- **Tokens de acceso.** Un token no está vinculado a un dispositivo. Por ese motivo no se diferencia su cantidad entre Pro y Max y ambos conservan el tope técnico de 10 tokens activos.
- **Topes de Max.** Son altos en lugar de ilimitados para acotar el costo máximo de una cuenta.
- **Excedentes.** Un usuario que queda por encima de un límite conserva lo que ya creó. Solo se impide crear recursos nuevos.
- **Interfaz.** Las funciones que el plan no permite no se muestran.

## 6. Precios y justificación frente a los costos

| Plan | Precio mensual | Importe en pesos |
| ---- | -------------- | ---------------- |
| Free | USD 0          | No aplica        |
| Pro  | USD 9          | ARS 14.100       |
| Max  | USD 29         | ARS 45.500       |

El importe en pesos surge del tipo de cambio oficial vendedor de ARS 1.570.

### 6.1 Referencia de mercado

Pro se ubica en la banda de entrada de las herramientas de colaboración con precio por usuario (Miro USD 8, Notion USD 10, Retool USD 10) y por debajo de los tableros de datos de precio fijo (Cyfe USD 29, Geckoboard USD 79). Max equivale a 3,2 veces Pro, dentro de la relación habitual entre el plan de entrada y el superior, y actúa como referencia que hace más atractivo a Pro.

### 6.2 Supuestos de ingresos

- El 3 % de los usuarios paga. La referencia publicada para productos con plan gratuito y contratación sin vendedor es de 3 a 5 %.
- De quienes pagan, el 80 % elige Pro y el 20 % Max. No existe un estudio primario sobre esta distribución y se declara como supuesto.
- Mercado Pago retiene 4,49 % más IVA con acreditación a 10 días, un 5,43 % efectivo.

### 6.3 Cobertura del costo en cada hito

Con 1.000 usuarios el pico es de 100 concurrentes. El 3 % son 30 pagadores: 24 en Pro y 6 en Max. El ingreso bruto es 24 × 9 + 6 × 29 = USD 390. Descontada la comisión de Mercado Pago quedan USD 369. El costo de infraestructura es de USD 47. El ingreso cubre el costo 7,8 veces.

El equilibrio se alcanza con 4 pagadores, 3 en Pro y 1 en Max, que aportan USD 53 netos. Equivale a una conversión del 0,4 %. Con una conversión pesimista del 1 %, 8 pagadores en Pro y 2 en Max aportan USD 123 netos y también cubren el costo.

La misma cuenta se repite en los demás hitos:

| Usuarios  | Pagadores en Pro y en Max | Ingreso neto y costo sobre ingreso |
| --------- | ------------------------- | ---------------------------------- |
| 1.000     | 24 y 6                    | 369 y 12,7 %                       |
| 10.000    | 240 y 60                  | 3.690 y 6,9 %                      |
| 100.000   | 2.400 y 600               | 36.900 y 4,4 %                     |
| 500.000   | 12.000 y 3.000            | 184.400 y 4,0 %                    |
| 1.000.000 | 24.000 y 6.000            | 368.800 y 2,4 %                    |
| 5.000.000 | 120.000 y 30.000          | 1.844.000 y 1,2 %                  |

Los dos últimos hitos incluyen el costo de operación de AWS.

### 6.4 Justificación del precio de cada plan

**Free.** Un usuario gratuito típico genera 3.200 solicitudes mensuales y cuesta entre USD 0,004 y 0,05 por mes según el hito. Con el tablero al límite (25 nodos, 300 segundos) y 8 horas diarias de uso cuesta USD 0,14. Cada pagador sostiene a 32 usuarios gratuitos, con un costo conjunto inferior a USD 1,60.

**Pro.** El peor caso es un tablero con 100 nodos, todos con fuente externa, abierto 8 horas por día hábil con el intervalo mínimo de 150 segundos. Genera 352.000 solicitudes mensuales, con un costo de USD 1,02 a la tarifa marginal de Vercel de USD 2,90 por millón de solicitudes. Sumados los usuarios gratuitos que sostiene, el costo máximo atribuible es de USD 2,60, frente a un ingreso neto de USD 8,51. El precio de USD 9 mantiene un margen superior al 69 % aun en el peor caso y permite un valor competitivo, acorde con el objetivo de concentrar en Pro a la mayoría de los pagadores.

**Max.** El peor caso es un tablero con 300 nodos con el intervalo mínimo de 60 segundos. Genera 2.100.000 solicitudes mensuales, con un costo de USD 6,10. Con los usuarios gratuitos asociados el costo máximo es de USD 7,70, frente a un ingreso neto de USD 27,43. Un precio inferior a USD 20 dejaría un margen insuficiente frente al uso intensivo que el plan habilita.

## 7. Cobro con Mercado Pago

- **Integración.** Se utiliza el producto Suscripciones sin plan asociado. La aplicación crea la suscripción en estado pendiente y redirige al usuario al sitio de Mercado Pago. No se almacenan ni procesan datos de tarjetas.
- **Confirmación.** Mercado Pago notifica cada cambio mediante un webhook firmado. La aplicación valida la firma, consulta el estado real de la suscripción y actualiza el plan del usuario.
- **Reintentos.** Ante un cobro rechazado Mercado Pago realiza hasta 4 reintentos en 10 días. Tras 3 cuotas rechazadas cancela la suscripción.
- **Cambios de plan.** No existe prorrateo. El aumento de plan rige de inmediato y la reducción al cierre del período.
- **Moneda.** Las cuentas argentinas cobran en pesos. El precio se define en dólares y el importe en pesos se actualiza con una modificación del monto de cada suscripción.
- **Comisiones.** 6,99 % con acreditación inmediata, 4,49 % a 10 días, 3,39 % a 18 días y 1,49 % a 35 días, más IVA. Se recomienda la acreditación a 35 días, con un costo efectivo del 1,80 %. En un ingreso recurrente la demora solo afecta al primer mes. Con esa opción el ingreso neto de cada hito aumenta un 3,8 %.
- **Alternativas.** Mercado Pago no tiene costo fijo. No se identificó una alternativa a la vez más económica y de integración igual o más simple. Los intermediarios que actúan como vendedor registrado cobran cerca del 5 % más USD 0,50 por operación. El débito directo en cuenta bancaria exige un convenio con el banco. El cobro queda aislado detrás de una interfaz, de modo que un cambio de proveedor afecta a un solo componente.
- **Impuestos.** Cada cobro requiere factura electrónica. Las retenciones de Ingresos Brutos dependen de la jurisdicción y de la condición fiscal y no están incluidas en las cifras.

## 8. Datos no verificados

- MongoDB no publica las operaciones por segundo de cada nivel dedicado. Los niveles indicados son estimaciones y requieren una prueba de carga.
- El costo de la migración a AWS proviene de fuentes con interés comercial. No se halló un caso independiente.
- La distribución de pagadores entre Pro y Max es un supuesto.
- Mercado Pago no documenta el monto mínimo de una suscripción ni el cobro en dólares para cuentas argentinas.
- Las tarifas de red del plan Pro de Redis Cloud no están publicadas.
- El nivel plano de red de Vercel declara que el tráfico excedente no se factura dentro de un uso razonable. Las cifras suponen el caso desfavorable de facturación por uso.

## 9. Fuentes

- Vercel: vercel.com/pricing, vercel.com/docs/functions/usage-and-pricing, vercel.com/docs/pricing/regional-pricing/iad1, vercel.com/docs/functions/limitations
- MongoDB Atlas: mongodb.com/pricing, mongodb.com/docs/atlas/reference/atlas-limits
- Redis Cloud: redis.io/pricing
- Resend: resend.com/pricing
- AWS: lista de precios pública de ECS, ELB, EC2, SES, ElastiCache y DocumentDB para us-east-1
- Mercado Pago: mercadopago.com.ar/developers, mercadopago.com.ar/herramientas-para-vender/suscripciones
- Conversión y precios de mercado: lennysnewsletter.com, chartmogul.com y las páginas de precios de cada producto citado
- Tipo de cambio: Banco de la Nación Argentina, 13 de noviembre de 2026
