
# ChronoNewsAPI

ChronoNewsAPI is a RESTful web service API designed for minimal news platform management. It is built using Golang and the Chi router. This API provides management of news posts, categories, and accounts, including the account reset process.

## Key Features

* **Asynchronous Image Handling**: To ensure a fast and responsive API, image uploads are handled asynchronously. The API receives an image, saves it, and immediately queues it for background processing without blocking the user's request.
* **Flexible Storage Support (S3 & Local)**: Supports saving media assets either to the local file system or to S3-compatible cloud object storage (e.g., AWS S3, Cloudflare R2). This ensures scalability for production environments while maintaining simplicity for development.
* **Decoupled Architecture**: Resource-intensive tasks like image compression (to WebP), file cleanup, and system maintenance are offloaded to a dedicated background worker, **[ChronoNewsScheduler](https://github.com/ScrKiddie/ChronoNewsScheduler)**. This separation of concerns keeps the API lightweight and highly available.
* **Dynamic Content Rebuilding**: The API dynamically injects processed image URLs (CDN or Local) back into the news content upon retrieval, ensuring that users always see the most up-to-date, optimized images without the API having to store large, pre-rendered content.
* **Comprehensive Management**: Provides complete CRUD (Create, Read, Update, Delete) operations for news posts, categories, and user accounts with role-based access control.

## Service Architecture

This project is part of a larger ecosystem and works in tandem with **[ChronoNewsScheduler](https://github.com/ScrKiddie/ChronoNewsScheduler)**, a dedicated background job processor.

* **ChronoNewsAPI (This Project)**: Acts as the primary interface for clients. It handles authentication, data validation, and management of database records. Its main responsibility is to remain fast and highly available.
* **[ChronoNewsScheduler](https://github.com/ScrKiddie/ChronoNewsScheduler)**: A robust Go-based worker that handles asynchronous tasks offloaded by the API. Its responsibilities include image compression to WebP, scheduled cleanup of old files, and recovering stuck tasks to ensure system reliability.

This separation of concerns allows the API to stay lightweight and responsive, while complex or long-running jobs are processed efficiently in the background.


## Technologies

* **Golang**: The primary programming language for developing ChronoNewsAPI.
* **Chi Router**: Used for routing in the web service.
* **GORM**: An ORM library for Golang to interact with the PostgreSQL database.
* **PostgreSQL**: Used as the database to store the application's data.
* **AWS SDK (S3)**: Used for interacting with S3-compatible storage services.


## Example Client Application

An example client application that consumes this API can be found here: **[ChronoNews](https://github.com/ScrKiddie/ChronoNews)**. This client demonstrates how to interact with the ChronoNewsAPI to build a complete news platform.

## Documentation

For the full API documentation, please refer to the documentation:
[ChronoNewsAPI Documentation](https://app.swaggerhub.com/apis-docs/ScrKiddy/ChronoNewsAPI/1.0.0)

## Environment Variables

The application requires specific environment variables to run. Below is the list of available configurations.

| **Key** | **Type** | **Description** | **Example** |
|---|---|---|---|
| **WEB\_BASE\_URL** | `string` | Public base URL of the API. Used for constructing absolute links. | `https://api.mydomain.com` |
| **WEB\_PORT** | `string` | Port on which the web service will run | `8080` |
| **WEB\_CORS\_ORIGINS** | `string` | List of allowed origins for Cross-Origin Resource Sharing (CORS) | `*,http://mydomain.com` |
| **WEB\_CLIENT\_URL** | `string` | Base URL of the frontend client application. | `http://localhost:3000` |
| **WEB\_CLIENT\_PATHS\_POST** | `string` | Client path for single post pages. | `/post` |
| **WEB\_CLIENT\_PATHS\_CATEGORY** | `string` | Client path for category pages. | `/category` |
| **WEB\_CLIENT\_PATHS\_RESET** | `string` | Client path for the password reset form. | `/reset-password` |
| **WEB\_CLIENT\_PATHS\_FORGOT** | `string` | Client path for the forgot password page. | `/forgot-password` |
| **JWT\_SECRET** | `string` | Secret key for JWT authentication | `mysecretkey12345` |
| **JWT\_EXP** | `integer` | Expiry time for the JWT token in hours | `24` |
| **CAPTCHA\_SECRET** | `string` | Secret key for Cloudflare Turnstile | `0x4AAAAAA...` |
| **DB\_USER** | `string` | Database user | `myuser` |
| **DB\_PASSWORD** | `string` | Database password | `mypassword` |
| **DB\_HOST** | `string` | Database host | `localhost` |
| **DB\_PORT** | `int` | Database port | `5432` |
| **DB\_NAME** | `string` | Database name | `mydatabase` |
| **DB\_SSLMODE** | `string` | SSL mode for database connection (`disable`, `require`, etc.) | `require` |
| **DB\_MIGRATION** | `boolean` | If `true`, runs database migrations on startup. | `false` |
| **RESET\_EXP** | `integer` | Expiry time for reset code in hours | `2` |
| **SMTP\_HOST** | `string` | SMTP server host | `smtp.example.com` |
| **SMTP\_PORT** | `integer` | SMTP server port | `587` |
| **SMTP\_USERNAME** | `string` | SMTP authentication username | `user123` |
| **SMTP\_PASSWORD** | `string` | SMTP authentication password | `pass123` |
| **SMTP\_FROM\_NAME** | `string` | Name of the sender for SMTP emails | `AppName` |
| **SMTP\_FROM\_EMAIL** | `string` | Email address of the sender for SMTP emails | `no-reply@domain.com` |
| **STORAGE\_MODE** | `string` | Storage mode: `local` or `s3` | `s3` |
| **STORAGE\_CDN\_URL** | `string` | Base URL for CDN (required if mode is `s3`, empty if `local`) | `https://cdn.mydomain.com` |
| **STORAGE\_PROFILE** | `string` | Directory path/prefix for profile pictures | `./storage/profile_picture/` |
| **STORAGE\_POST** | `string` | Directory path/prefix for post pictures | `./storage/post_picture/` |
| **STORAGE\_S3\_BUCKET** | `string` | S3 Bucket Name (Required if mode is `s3`) | `my-bucket` |
| **STORAGE\_S3\_REGION** | `string` | S3 Region (e.g., `auto` for R2, `us-east-1` for AWS) | `auto` |
| **STORAGE\_S3\_ACCESS\_KEY** | `string` | S3 Access Key ID | `access_key` |
| **STORAGE\_S3\_SECRET\_KEY** | `string` | S3 Secret Access Key | `secret_key` |
| **STORAGE\_S3\_ENDPOINT** | `string` | S3 Endpoint URL (Required for Cloudflare R2 / MinIO) | `https://<id>.r2.cloudflarestorage.com` |

### Configuration for Testing

The application uses a separate, minimal configuration for running tests. Most test configurations are loaded from environment variables prefixed with `TEST_`, but several key sections are **hardcoded** for simplicity and consistency.

**Hardcoded Test Configurations (No Env Vars Needed):**

*   **`storage`**: This entire section is **removed** from the test configuration. Tests automatically use a temporary local directory for file storage, which is created and deleted on the fly.
*   **`reset`**: The reset token expiration (`reset.exp`) is hardcoded to `2` hours.
*   **`web.client_url` & `web.client_paths`**: The client-facing URLs are hardcoded to mock values (e.g., `http://test-client.com/post`).

**Environment Variables for Testing:**

*   **`captcha`**: The captcha configuration for tests is different. Instead of a single secret, you must provide three specific keys to mock different outcomes:
    ```bash
    TEST_CAPTCHA_SECRET_PASS="1x0000000000000000000000000000000AA"
    TEST_CAPTCHA_SECRET_FAIL="2x0000000000000000000000000000000AA"
    TEST_CAPTCHA_SECRET_USAGE="3x0000000000000000000000000000000AA"
    ```
*   **`jwt`**, **`db`**, **`smtp`**, and parts of **`web`** still need to be provided with the `TEST_` prefix (e.g., `TEST_JWT_SECRET`, `TEST_DB_HOST`).

### JSON Configuration (`config.json`)

If you use a `config.json` file, be aware that the structure for the `test` object is different from the main configuration. It omits the `storage` and `reset` sections, and does not require client URL paths. Please refer to `config.example.json` for the exact structure.
