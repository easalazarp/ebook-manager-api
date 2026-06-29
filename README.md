# E-Book Management Backend

Aplicación web full-stack monolítica en Go para gestionar una biblioteca digital de e-books. Permite a usuarios anónimos navegar el catálogo y a usuarios autenticados descargar libros. Los administradores gestionan el catálogo a través de una API REST y un panel de administración.

---

## Stack Tecnológico

| Tecnología             | Rol                                             |
| ---------------------- | ----------------------------------------------- |
| **Go 1.22+**           | Lenguaje de programación                        |
| **Chi v5**             | Router HTTP                                     |
| **PostgreSQL**         | Base de datos relacional (hosteada en Supabase) |
| **pgxpool (pgx v5)**   | Driver PostgreSQL con pool de conexiones        |
| **Supabase Auth**      | Autenticación de usuarios (JWT ES256 via JWKS)  |
| **Supabase Storage**   | Almacenamiento de archivos PDF/EPUB y portadas  |
| **html/template**      | Motor de plantillas para las vistas MVC         |
| **Tailwind CSS (CDN)** | Estilos de la interfaz gráfica                  |
| **lestrrat-go/jwx v2** | Validación de tokens JWT ES256 via JWKS         |
| **google/uuid**        | Generación de IDs únicos (UUID v4)              |
| **swaggo/swag**        | Documentación de API con Swagger/OpenAPI        |

---

## Arquitectura

El proyecto sigue el patrón **MVC en capas**. Cada capa solo conoce a la inmediatamente inferior a través de **interfaces**, sin instanciar implementaciones concretas directamente. Todas las dependencias se inyectan desde `cmd/server/main.go`.

```
┌──────────────────────────────────────────┐
│           CLIENTE (Navegador)            │
└─────────────────┬────────────────────────┘
                  │ HTTP (HTML o JSON)
┌─────────────────▼────────────────────────┐
│          CAPA DE PRESENTACIÓN            │
│  Handlers HTTP + Vistas html/template    │
│  (internal/handlers + internal/views)    │
└─────────────────┬────────────────────────┘
                  │ Interfaces de Servicio
┌─────────────────▼────────────────────────┐
│           CAPA DE NEGOCIO                │
│            (internal/service)            │
└─────────────────┬────────────────────────┘
                  │ Interfaces de Repositorio
┌─────────────────▼────────────────────────┐
│            CAPA DE DATOS                 │
│          (internal/repository)           │
└─────────────────┬────────────────────────┘
                  │ pgxpool — SQL parametrizado
┌─────────────────▼────────────────────────┐
│       PostgreSQL en Supabase (nube)      │
└──────────────────────────────────────────┘
```

---

## Estructura del Proyecto

```
e-book/
├── cmd/server/
│   └── main.go                             # Punto de entrada: ensambla todas las dependencias
├── internal/
│   ├── config/
│   │   └── config.go                       # Lectura y validación de variables de entorno
│   ├── middleware/
│   │   └── auth.go                         # Middleware JWT ES256: valida JWKS y protege rutas
│   ├── models/
│   │   ├── base.go                         # BaseModel: ID, CreatedAt, UpdatedAt
│   │   ├── ebook.go                        # EBook, EBookFile, validación de formato
│   │   ├── author.go                       # Author con getters encapsulados
│   │   ├── category.go                     # Category
│   │   └── log.go                          # Log de auditoría
│   ├── repository/
│   │   ├── errors.go                       # ErrNotFound (error centinela)
│   │   ├── *_repository.go                 # Interfaces de repositorio
│   │   └── postgres_*_repository.go        # Implementaciones PostgreSQL
│   ├── service/
│   │   ├── *_service.go                    # Interfaces + DTOs de servicio
│   │   └── *_service_impl.go               # Implementaciones de servicios
│   ├── storage/
│   │   └── supabase_storage.go             # Integración con Supabase Storage
│   ├── handlers/
│   │   ├── render.go                       # Helper render() para templates HTML
│   │   ├── response.go                     # Helpers writeJSON() y writeError()
│   │   ├── ebook_handler.go                # API REST para e-books
│   │   ├── author_handler.go               # API REST para autores
│   │   ├── category_handler.go             # API REST para categorías
│   │   ├── auth_handler.go                 # MVC: login / register / logout
│   │   ├── auth_api_handler.go             # API REST de autenticación
│   │   ├── web_handler.go                  # MVC: catálogo, detalle, descarga
│   │   ├── admin_handler.go                # Panel de administración (MVC)
│   │   └── upload_handler.go               # Subida de archivos a Supabase Storage
│   └── views/
│       ├── layout.html                     # Plantilla base con navbar
│       ├── catalog.html                    # Vista: catálogo de e-books
│       ├── authors.html                    # Vista: lista de autores
│       ├── detail.html                     # Vista: detalle de un e-book
│       ├── login.html                      # Vista: formulario de login
│       ├── register.html                   # Vista: formulario de registro
│       ├── admin_dashboard.html            # Panel admin: resumen
│       ├── admin_create_ebook.html         # Panel admin: crear e-book
│       ├── admin_edit_ebook.html           # Panel admin: editar e-book
│       ├── admin_create_author.html        # Panel admin: crear autor
│       ├── admin_edit_author.html          # Panel admin: editar autor
│       ├── admin_create_category.html      # Panel admin: crear categoría
│       ├── admin_edit_category.html        # Panel admin: editar categoría
│       └── admin_logs.html                 # Panel admin: logs de auditoría
├── docs/
│   ├── docs.go                             # Documentación Swagger generada
│   ├── swagger.json
│   └── swagger.yaml
├── .env.example                            # Plantilla de variables de entorno
├── go.mod
└── go.sum
```

---

## Requisitos Previos

- [Go 1.22+](https://go.dev/dl/)
- Una cuenta en [Supabase](https://supabase.com/) con un proyecto activo
- Una base de datos PostgreSQL configurada en dicho proyecto

---

## Configuración

### 1. Clonar el repositorio

```bash
git clone https://github.com/yourorg/ebook-management-backend.git
cd ebook-management-backend
```

### 2. Instalar dependencias

```bash
go mod tidy
```

### 3. Configurar variables de entorno

Copia el archivo de ejemplo y completa los valores con los de tu proyecto Supabase:

```bash
cp .env.example .env
```

| Variable                    | Descripción                                                                 | Requerida              |
| --------------------------- | --------------------------------------------------------------------------- | ---------------------- |
| `PORT`                      | Puerto HTTP del servidor                                                    | No (default: `8080`)   |
| `DATABASE_URL`              | URL de conexión PostgreSQL (puerto 6543 para PgBouncer)                     | **Sí**                 |
| `SUPABASE_JWT_SECRET`       | Secret JWT del proyecto (referencia, no usado para validar)                 | **Sí**                 |
| `SUPABASE_URL`              | URL base del proyecto Supabase (`https://xxxx.supabase.co`)                 | **Sí**                 |
| `SUPABASE_ANON_KEY`         | Clave anónima para llamar a la Auth API                                     | **Sí**                 |
| `SUPABASE_SERVICE_ROLE_KEY` | Clave de servicio para operaciones de Storage. **Nunca exponer al cliente** | **Sí**                 |
| `SUPABASE_BUCKET`           | Nombre del bucket en Supabase Storage                                       | No (default: `ebooks`) |

> **Nota sobre DATABASE_URL:** Supabase usa PgBouncer en transaction pooling mode. Usa el puerto `6543` y el driver está configurado con `SimpleProtocol` para compatibilidad.

### 4. Ejecutar el servidor

```bash
go run ./cmd/server/main.go
```

El servidor estará disponible en `http://localhost:8080`.

---

## Endpoints

### API REST

Todos los endpoints de la API requieren autenticación via header `Authorization: Bearer <token>` o cookie `token`.

| Método   | Ruta                         | Descripción                      |
| -------- | ---------------------------- | -------------------------------- |
| `GET`    | `/health`                    | Health check — `{"status":"ok"}` |
| `POST`   | `/api/v1/ebooks`             | Crear e-book                     |
| `GET`    | `/api/v1/ebooks`             | Listar todos los e-books         |
| `GET`    | `/api/v1/ebooks/{id}`        | Obtener e-book por ID            |
| `PUT`    | `/api/v1/ebooks/{id}`        | Actualización parcial de e-book  |
| `DELETE` | `/api/v1/ebooks/{id}`        | Eliminar e-book                  |
| `POST`   | `/api/v1/authors`            | Crear autor                      |
| `GET`    | `/api/v1/authors`            | Listar autores                   |
| `GET`    | `/api/v1/ebooks/{id}/author` | Obtener el autor de un e-book    |

### Web MVC (HTML)

| Método | Ruta                    | Auth   | Descripción                            |
| ------ | ----------------------- | ------ | -------------------------------------- |
| `GET`  | `/`                     | No     | Catálogo de e-books                    |
| `GET`  | `/authors`              | No     | Lista de autores                       |
| `GET`  | `/ebooks/{id}`          | No     | Detalle del e-book                     |
| `GET`  | `/ebooks/{id}/download` | **Sí** | Descarga (redirect a Supabase Storage) |
| `GET`  | `/login`                | No     | Formulario de login                    |
| `POST` | `/login`                | No     | Procesa login via Supabase Auth        |
| `GET`  | `/register`             | No     | Formulario de registro                 |
| `POST` | `/register`             | No     | Procesa registro via Supabase Auth     |
| `POST` | `/logout`               | No     | Cierra sesión (borra cookie)           |

### Documentación Swagger

Una vez el servidor esté corriendo, la documentación interactiva de la API está disponible en:

```
http://localhost:8080/swagger/index.html
```

---

## Autenticación

Supabase firma sus tokens JWT con **ES256** (ECDSA P-256). Al arrancar, el servidor descarga las claves públicas del endpoint JWKS de Supabase:

```
GET {SUPABASE_URL}/auth/v1/.well-known/jwks.json
```

Cada request entrante valida la firma del token contra estas claves públicas. El token se acepta en cookie `HttpOnly` (`token`) o en el header `Authorization: Bearer`.

- **Rutas MVC:** redirigen a `/login` (302) si el token falta o es inválido.
- **Rutas API:** responden `401 JSON` si el token falta o es inválido.

---

## Patrones de Diseño

| Patrón                 | Aplicación                                                             |
| ---------------------- | ---------------------------------------------------------------------- |
| Repository Pattern     | Desacopla la lógica de negocio del motor de base de datos              |
| Service Layer          | Centraliza la lógica de negocio                                        |
| Dependency Injection   | `main.go` inyecta todas las dependencias vía interfaces                |
| DTO                    | `CreateEBookInput`, `UpdateEBookInput` separan entrada de dominio      |
| Partial Update         | `UpdateEBookInput` usa punteros `*string` para no sobreescribir vacíos |
| Sentinel Error         | `repository.ErrNotFound` detectado con `errors.Is`                     |
| Typed Error            | `models.ErrInvalidFormat` detectado con `errors.As`                    |
| Error Wrapping         | `fmt.Errorf("contexto: %w", err)` en repositorios para trazabilidad    |
| Compile-time assertion | `var _ IFoo = (*Impl)(nil)` detecta interfaces incompletas al compilar |
| MVC                    | Handlers (C) + Views html/template (V) + Models (M)                    |

---

## Seguridad

- **JWT ES256 via JWKS:** validación con clave pública ECDSA. Resistente a ataques de confusión de algoritmo.
- **HttpOnly Cookie:** el token no es accesible desde JavaScript, previniendo XSS.
- **SameSite=Lax:** previene ataques CSRF en la mayoría de los escenarios.
- **SQL parametrizado:** todas las queries usan `$1, $2, ...` con pgx. Sin concatenación de strings (previene SQL Injection).
- **Timeout de 30s:** el middleware `chi.Timeout` evita que conexiones lentas agoten recursos.
- **Recoverer:** captura panics en handlers y retorna HTTP 500 sin crashear el servidor.
- **Service Role Key:** solo se usa en el servidor para operaciones de Storage. Nunca se expone al cliente.
