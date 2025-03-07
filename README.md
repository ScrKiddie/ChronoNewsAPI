# ChronoVerseAPI
ChronoVerseAPI is a RESTful web service API designed for minimal news platform management. It is built using Golang and the Chi router. This API provides efficient and parallel processing of news content, including automatic detection, processing, compression, validation, and storage of image files.

## Key Features
The main feature of ChronoVerseAPI is its ability to automatically detect and process base64 encoded images in the `img` tag of news content requests. The API performs automatic compression, validation, and stores the files in parallel, ensuring efficiency and minimizing processing time.

## Technologies
* **Golang**: The primary programming language for developing ChronoVerseAPI.
* **Chi Router**: Used for routing in the web service.
* **GORM**: An ORM library for Golang to interact with the PostgreSQL database.
* **PostgreSQL**: Used as the database to store the application's data.

## Environment Variables
| **Key**                | **Type**     | **Description**                                                                       | **Example**       |
|--------------------|----------|---------------------------------------------------------------------------------------|-------------------|
| **JWT_SECRET**     | `string` | Secret key for JWT authentication                                                        | `YOUR_SECRET_JWT` |
| **JWT_EXP**     | `integer` | Expiry time for JWT token in hours | `24`              |
| **WEB_PORT**     | `integer` | Port on which the web service will run | `3000`            |
| **WEB_CORS_ORIGINS**     | `string` | List of allowed origins for Cross-Origin Resource Sharing (CORS) | `*,http://example.com,http://anotherdomain.com`            |
| **CAPTCHA_SECRET** | `string` | Secret key untuk Cloudflare Turnstile.                                           | `0x4AAAAAAABBBBCCCCDDDD1234567890EE` |
| **DB_USERNAME**    | `string` | Database username                                                          | `postgres`                                      |
| **DB_PASSWORD**    | `string` | Database password	postg                                                          | `password123`                               |
| **DB_HOST**        | `string` | Database host                                                                 | `localhost`                                 |
| **DB_PORT**        | `int`    | Database port                                                        | `3306`                                      |
| **DB_NAME**        | `string` | Database name                                                                | `prbcare`                                   |
| **STORAGE_PROFILE**        | `string` | Path where profile pictures are stored                                                                | `./storage/profile_picture/`                                   |
| **STORAGE_POST**        | `string` | Path where post pictures are stored                                                                | `./storage/post_picture/`                                   |
## Documentation
For the full API documentation, please refer to the SwaggerHub documentation:
[ChronoVerseAPI Documentation](http://app.swaggerhub.com/apis-docs/scrkiddie/ChronoVerseAPI/1.0.0)