
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

The application requires specific environment variables to run. Below is the list of available configurations, including the newly added S3 settings.


| **Key** | **Type** | **Description** | **Example** |
|---|---|---|---|
| **WEB\_BASE\_URL** | `string` |  Public base URL of the API. Used for constructing absolute links. | `https://api.mydomain.com` |
| **WEB\_PORT** | `string` | Port on which the web service will run | `8080` |
| **WEB\_CORS\_ORIGINS** | `string` | List of allowed origins for Cross-Origin Resource Sharing (CORS) | `*,http://mydomain.com` |
| **JWT\_SECRET** | `string` | Secret key for JWT authentication | `mysecretkey12345` |
| **JWT\_EXP** | `integer` | Expiry time for the JWT token in hours | `24` |
| **CAPTCHA\_SECRET** | `string` | Secret key for Cloudflare Turnstile | `0x4AAAAAA...` |
| **DB\_USER** | `string` | Database user | `myuser` |
| **DB\_PASSWORD** | `string` | Database password | `mypassword` |
| **DB\_HOST** | `string` | Database host | `localhost` |
| **DB\_PORT** | `int` | Database port | `5432` |
| **DB\_NAME** | `string` | Database name | `mydatabase` |
| **RESET\_URL** | `string` | URL for password reset (Frontend) | `http://mydomain.com/reset` |
| **RESET\_QUERY** | `string` | Query parameter for password reset | `code` |
| **RESET\_REQUEST\_URL** | `string` | URL for password reset request (Frontend) | `http://mydomain.com/forgot` |
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

### Variable Prefix & Structural Differences

The application distinguishes between standard and test configurations based on the environment variable keys. Crucially, **the Test configuration is a minimized subset** of the main configuration.
#### 1. Standard Configuration (No Prefix)
Variables *without* a prefix (e.g., `WEB_BASE_URL`, `STORAGE_MODE`) are loaded into the main application configuration.

#### 2. Test Configuration (`TEST_` Prefix)
Variables *starting with* `TEST_` are loaded into a separate, smaller `TestConfig` structure. Because the struct is different, **many keys available in production do not exist in the test configuration.**

| Config Section | Standard Config (Full) | Test Config (Subset) | **Implication for Env Vars** |
| :--- | :--- | :--- | :--- |
| **Web** | `base_url`, `port`, `cors_origins` | `port`, `cors_origins` | `TEST_WEB_BASE_URL` **does not exist** and is ignored. |
| **Storage** | Full Support (`local` + `s3`) | **Removed Completely** | `TEST_STORAGE_*` keys **do not exist**. Storage is hardcoded to temporary local dirs. |
| **Captcha** | Single String (`secret`) | Object (`pass`, `fail`, `usage`) | Requires 3 separate keys for testing (see below). |

### Storage Behavior in Testing

Since the `storage` configuration block is structurally absent from the Test Config:

* **Production/Dev:** Fully configurable via `STORAGE_MODE` (Local or S3).
* **Test Environment:** **No configuration possible.** The test suite automatically creates and tears down a temporary local directory. Any attempt to set S3 credentials for testing via Env Vars will be ignored because there are no fields to hold them.

### Captcha in Testing

* **Standard:** Uses `CAPTCHA_SECRET`.
* **Test:** Uses a structured object to mock different Turnstile outcomes. You must provide these specific keys:
  ```bash
  TEST_CAPTCHA_SECRET_PASS="1x0000000000000000000000000000000AA"
  TEST_CAPTCHA_SECRET_FAIL="2x0000000000000000000000000000000AA"
  TEST_CAPTCHA_SECRET_USAGE="3x0000000000000000000000000000000AA"
  ```

### JSON Configuration (`config.json`)

If using `config.json` instead of environment variables, please note the schema difference. The `test` object is **not** a copy of the root object; it lacks `storage` and `web.base_url`. Refer to `config.example.json` for the exact structure.
