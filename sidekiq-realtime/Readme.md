
# Sidekiq realtime servide

A brief description of what this project does and who it's for


## Run Locally

Clone the project

Go to the project directory

```bash
  cd sidekiq-realtime
```

Install dependencies

```bash
  npm install 
```
OR
```bash
  make install 
```

Start the server

```bash
  npm run start
```
OR 

```bash
  make run
```


## Environment Variables

To run this project, you will need to add the following environment variables to your .env file

`GRPC_BIND` : Url where server will start. `default: 127.0.0.1:50051`

