FROM golang:1.18.3 as builder

RUN mkdir /builder
COPY . /build/
WORKDIR /build

# RUN go build -o main .
# RUN  apk add build-base

    
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o main .

FROM golang:1.18.3

COPY --from=builder /build/main /app/
COPY ./apns /app/apns
COPY ./storage/mime.types /etc/
WORKDIR /app

ENV PATH="/app:${PATH}"

ADD https://sidekiqconfig.s3.amazonaws.com/default.yaml .

RUN ls

RUN apt update && apt upgrade -y

# install aws cli
RUN apt install zip -y &&  \
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip" && \
  unzip awscliv2.zip && \
  ./aws/install && \
  apt install vim -y

# setup aws and link it to wasabi:
RUN rm -rf ~/.aws && \
  mkdir ~/.aws && \
  touch ~/.aws/config && \
  printf "[profile wasabi]\nendpoint_url = http://s3.us-east-2.wasabisys.com\nregion = us-east-2\noutput=json" > ~/.aws/config && \
  touch ~/.aws/credentials && \
  printf "[wasabi]\naws_access_key_id = 72I5N34INHRZFSNE9U3Q\naws_secret_access_key = pQIWZEoQQK38SjzCwh8slsacDVT6DkWlBmeswgTe" \n > ~/.aws/credentials 

RUN echo export AWS_PROFILE=wasabi >> ~/.bashrc

CMD ["./main" , "serve"]
