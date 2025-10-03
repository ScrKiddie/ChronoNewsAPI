# ChronoNewsAPI
ChronoNewsAPI is a RESTful web service API designed for minimal news platform management. It is built using Golang and the Chi router. This API provides management of news posts, categories, and accounts, including the account reset process.

## Key Features
*   **Asynchronous Image Handling**: To ensure a fast and responsive API, image uploads are handled asynchronously. The API receives an image, saves it, and immediately queues it for background processing without blocking the user's request.
*   **Decoupled Architecture**: Resource-intensive tasks like image compression (to WebP), file cleanup, and system maintenance are offloaded to a dedicated background worker, **[ChronoNewsScheduler](https://github.com/ScrKiddie/ChronoNewsScheduler)**. This separation of concerns keeps the API lightweight and highly available.
*   **Dynamic Content Rebuilding**: The API dynamically injects processed image URLs back into the news content upon retrieval, ensuring that users always see the most up-to-date, optimized images without the API having to store large, pre-rendered content.
*   **Comprehensive Management**: Provides complete CRUD (Create, Read, Update, Delete) operations for news posts, categories, and user accounts with role-based access control.


## Service Architecture
This project is part of a larger ecosystem and works in tandem with **[ChronoNewsScheduler](https://github.com/ScrKiddie/ChronoNewsScheduler)**, a dedicated background job processor.

*   **ChronoNewsAPI (This Project)**: Acts as the primary interface for clients. It handles authentication, data validation, and management of database records. Its main responsibility is to remain fast and highly available.
*   **[ChronoNewsScheduler](https://github.com/ScrKiddie/ChronoNewsScheduler)**: A robust Go-based worker that handles asynchronous tasks offloaded by the API. Its responsibilities include image compression to WebP, scheduled cleanup of old files, and recovering stuck tasks to ensure system reliability.

This separation of concerns allows the API to stay lightweight and responsive, while complex or long-running jobs are processed efficiently in the background.

## Example Client Application
An example client application that consumes this API can be found here: **[ChronoNews](https://github.com/ScrKiddie/ChronoNews)**. This client demonstrates how to interact with the ChronoNewsAPI to build a complete news platform.


## Technologies
* **Golang**: The primary programming language for developing ChronoNewsAPI.
* **Chi Router**: Used for routing in the web service.
* **GORM**: An ORM library for Golang to interact with the PostgreSQL database.
* **PostgreSQL**: Used as the database to store the application's data.

## Environment Variables
| **Key**                     | **Type**  | **Description**                                                                                       | **Example**                                      |
|-----------------------------|-----------|-------------------------------------------------------------------------------------------------------|--------------------------------------------------|
| **JWT_SECRET**              | `string`  | Secret key for JWT authentication                                                                    | `mysecretkey12345`                               |
| **JWT_EXP**                 | `integer` | Expiry time for the JWT token in hours                                                                  | `24`                                             |
| **WEB_PORT**                | `string`  | Port on which the web service will run                                                                  | `8080`                                           |
| **WEB_CORS_ORIGINS**        | `string`  | List of allowed origins for Cross-Origin Resource Sharing (CORS)                                        | `*,http://mydomain.com,http://anotherdomain.com` |
| **CAPTCHA_SECRET**          | `string`  | Secret key for Cloudflare Turnstile                                                                   | `abcdef1234567890abcdef1234567890`               |
| **DB_USER**             | `string`  | Database user                                                                                     | `myuser`                                         |
| **DB_PASSWORD**             | `string`  | Database password                                                                                     | `mypassword`                                     |
| **DB_HOST**                 | `string`  | Database host                                                                                         | `localhost`                                      |
| **DB_PORT**                 | `int`     | Database port                                                                                         | `5432`                                           |
| **DB_NAME**                 | `string`  | Database name                                                                                         | `mydatabase`                                     |
| **RESET_URL**               | `string`  | URL for password reset                                                                                 | `http://mydomain.com/reset`                      |
| **RESET_QUERY**             | `string`  | Query parameter for password reset                                                                     | `code`                                           |
| **RESET_REQUEST_URL**       | `string`  | URL for password reset request                                                                         | `http://mydomain.com/forgot`                     |
| **RESET_EXP**               | `integer` | Expiry time for reset code in hours                                                                     | `2`                                              |
| **SMTP_HOST**               | `string`  | SMTP server host                                                                                        | `smtp.example.com`                               |
| **SMTP_PORT**               | `integer` | SMTP server port                                                                                        | `587`                                            |
| **SMTP_USERNAME**           | `string`  | SMTP authentication username                                                                            | `dummyusername123`                               |
| **SMTP_PASSWORD**           | `string`  | SMTP authentication password                                                                            | `dummypassword123`                               |
| **SMTP_FROM_NAME**          | `string`  | Name of the sender for SMTP emails                                                                      | `AppName`                                        |
| **SMTP_FROM_EMAIL**         | `string`  | Email address of the sender for SMTP emails                                                             | `dummyemail@domain.com`                          |
| **STORAGE_PROFILE**         | `string`  | Path where profile pictures are stored                                                                   | `./storage/profile_picture/`                     |
| **STORAGE_POST**            | `string`  | Path where post pictures are stored                                                                     | `./storage/post_picture/`                        |

## Environment Configuration

This application utilizes different configuration structures for **standard** (production/development) and **test** environments. This approach allows for more granular control during automated testing.

### JSON Configuration (`config.json`)

For details on how to structure the `config.json` file for different environments, please refer to the `config.example.json` file in the root directory.

### Environment Variables

When using environment variables, test-specific keys are prefixed with `TEST_`. Nested structures are flattened by joining keys with an underscore (`_`).

Essentially, all environment variables for the **test** mode mirror the **standard** ones but with a `TEST_` prefix. The main exception is `CAPTCHA_SECRET`, which is uniquely split into three modes (`_PASS`, `_FAIL`, `_USAGE`) to facilitate different testing scenarios.

#### **Standard Configuration**

A single environment variable defines the secret.

```bash
CAPTCHA_SECRET="YOUR_REAL_SECRET_KEY"
```

#### **Test Configuration**

Multiple, specific variables are used, each corresponding to a different test outcome.

```bash
TEST_CAPTCHA_SECRET_PASS="CLOUDFLARE_TEST_PASS_KEY"
TEST_CAPTCHA_SECRET_FAIL="CLOUDFLARE_TEST_FAIL_KEY"
TEST_CAPTCHA_SECRET_USAGE="CLOUDFLARE_TEST_USAGE_KEY"
```


## Documentation
For the full API documentation, please refer to the documentation:
[ChronoNewsAPI Documentation](https://app.swaggerhub.com/apis-docs/ScrKiddy/ChronoNewsAPI/1.0.0)