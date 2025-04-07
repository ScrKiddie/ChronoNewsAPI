# ChronoNewsAPI
ChronoNewsAPI is a RESTful web service API designed for minimal news platform management. It is built using Golang and the Chi router. This API provides management of news posts, categories, and accounts, including the account reset process.

## Key Features
The main feature of ChronoNewsAPI is its ability to automatically detect and process base64 encoded images in the `img` tag of news `content` requests. The API performs automatic compression, validation, and stores the files in parallel, ensuring efficiency and minimizing processing time.

## Technologies
* **Golang**: The primary programming language for developing ChronoNewsAPI.
* **Chi Router**: Used for routing in the web service.
* **GORM**: An ORM library for Golang to interact with the PostgreSQL database.
* **PostgreSQL**: Used as the database to store the application's data.

## Environment Variables
| **Key**                     | **Type**     | **Description**                                                                                       | **Example**                                      |
|-----------------------------|--------------|-------------------------------------------------------------------------------------------------------|--------------------------------------------------|
| **JWT_SECRET**              | `string`     | Secret key for JWT authentication                                                                    | `mysecretkey12345`                               |
| **JWT_EXP**                 | `integer`    | Expiry time for the JWT token in hours                                                                  | `24`                                             |
| **WEB_PORT**                | `integer`    | Port on which the web service will run                                                                  | `8080`                                           |
| **WEB_CORS_ORIGINS**        | `string`     | List of allowed origins for Cross-Origin Resource Sharing (CORS)                                        | `*,http://mydomain.com,http://anotherdomain.com` |
| **CAPTCHA_SECRET**          | `string`     | Secret key for Cloudflare Turnstile                                                                   | `abcdef1234567890abcdef1234567890`               |
| **DB_USERNAME**             | `string`     | Database username                                                                                     | `myuser`                                         |
| **DB_PASSWORD**             | `string`     | Database password                                                                                     | `mypassword`                                     |
| **DB_HOST**                 | `string`     | Database host                                                                                         | `localhost`                                      |
| **DB_PORT**                 | `int`        | Database port                                                                                         | `5432`                                           |
| **DB_NAME**                 | `string`     | Database name                                                                                         | `mydatabase`                                     |
| **RESET_URL**               | `string`     | URL for password reset                                                                                 | `http://mydomain.com/reset`                      |
| **RESET_QUERY**             | `string`     | Query parameter for password reset                                                                     | `code`                                           |
| **RESET_REQUEST_URL**       | `string`     | URL for password reset request                                                                         | `http://mydomain.com/forgot`                     |
| **RESET_EXP**               | `integer`    | Expiry time for reset code in hours                                                                     | `2`                                              |
| **SMTP_HOST**               | `string`     | SMTP server host                                                                                        | `smtp.example.com`                               |
| **SMTP_PORT**               | `integer`    | SMTP server port                                                                                        | `587`                                            |
| **SMTP_USERNAME**           | `string`     | SMTP authentication username                                                                            | `dummyusername123`                               |
| **SMTP_PASSWORD**           | `string`     | SMTP authentication password                                                                            | `dummypassword123`                               |
| **SMTP_FROM_NAME**          | `string`     | Name of the sender for SMTP emails                                                                      | `AppName`                                        |
| **SMTP_FROM_EMAIL**         | `string`     | Email address of the sender for SMTP emails                                                             | `dummyemail@domain.com`                          |
| **STORAGE_PROFILE**         | `string`     | Path where profile pictures are stored                                                                   | `./storage/profile_picture/`                     |
| **STORAGE_POST**            | `string`     | Path where post pictures are stored                                                                     | `./storage/post_picture/`                        |

## Documentation
For the full API documentation, please refer to the documentation:
[ChronoNewsAPI Documentation](https://app.swaggerhub.com/apis-docs/ScrKiddy/ChronoNewsAPI/1.0.0)