# Receipt Processor
### A simple tool to process JSON receipts

<br>

## Getting Started
Assuming you already have Go and Docker installed, this project should be relatively
easy to boot up. The repository depends on "docker-compose" for container orchestration.
If you installed Docker Desktop on Windows or Mac to procure docker, then you already
have docker-compose as it comes with the Docker Desktop installation.

Note: `text like this` means these are commands to be entered into your terminal.

1. Check if you already have docker-compose installed
    1. `docker-compose --version`
    2. If the output to the above is similar to "Docker Compose version..." then you're all set. If not, or if the version number is v1.3 or less, jump to section 'Installing docker-compose'.
    3. If you have docker-compose >v1.3, run the following command in the same level of the project as the docker-compose.yml file:
    `docker-compose up --build -d`
    4. The API is now booted up on localhost:8080. Refer to section 'Example cURL commands' for some ways you can test the API.
    5. You can check container status with `docker ps -a` and logs with `docker logs <container-name>`
2. Installing docker-compose
    1. If you are on Windows/Mac, the quickest and simplest way to procure docker-compose is to install Docker Desktop from the official Docker page: https://www.docker.com/products/docker-desktop/ . Boot it up and check `docker-compose --version`.
    2. Alternatively if you are using Homebrew as a native package manager, you can run `brew install docker-compose`.
    3. If you are on a Linux distribution you can download the binary from the docker-compose group's github releases page: https://github.com/docker/compose/releases . Follow the instructions under 'Where to Get Docker Compose' at this page: https://github.com/docker/compose
    4. After confirming docker-compose is installed by running `docker-compose --version` you can boot up the API by running `docker-compose up --build -d` at the same level as docker-compose.yml (top level of the project folder).

## Example cURL commands (if you're c/p'ing from the .md file don't include the backticks ``)
1. `curl -X POST http://localhost:8080/receipts/process -H "Content-Type: application/json" -d '{ "retailer": "M&M Corner Market", "purchaseDate": "2022-03-20", "purchaseTime": "14:33", "items": [ { "shortDescription": "Gatorade", "price": "2.25" },{ "shortDescription": "Gatorade", "price": "2.25" },{ "shortDescription": "Gatorade", "price": "2.25" },{ "shortDescription": "Gatorade", "price": "2.25" } ], "total": "9.00" }'`
2. `curl -X POST http://localhost:8080/receipts/process -H "Content-Type: application/json" -d '{ "retailer": "Target", "purchaseDate": "2022-01-01", "purchaseTime": "13:01", "items": [ { "shortDescription": "Mountain Dew 12PK", "price": "6.49" },{ "shortDescription": "Emils Cheese Pizza", "price": "12.25" },{ "shortDescription": "Knorr Creamy Chicken", "price": "1.26" },{ "shortDescription": "Doritos Nacho Cheese", "price": "3.35" },{ "shortDescription": " Klarbrunn 12-PK 12 FL OZ ", "price": "12.00" } ], "total": "35.35" }'`
3. `curl http://localhost:8080/receipts/{id}/points` (keep in mind there's a 10 minute TTL on the Redis setter, if you'd like to remove this set REDIS_TTL_IN_S=0 in docker-compose.yml)

## Author's Notes
All in all this was a fun project and a good opportunity for me to practice some of the Go skills I've been developing over the last few months. If I had more time or if this were truly a production environment I might've set up nginx and SSL, a logger better than go std "log" for multi-level logging, and I would've properly managed secrets with a .env or secrets manager rather than hard coding them into docker-compose.yml.

Thanks for checking my work out!
