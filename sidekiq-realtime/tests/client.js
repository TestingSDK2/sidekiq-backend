const grpc = require("@grpc/grpc-js");
var protoLoader = require("@grpc/proto-loader");
const PROTO_PATH = "../protobuf/v1/delivery.proto";

const options = {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
};

var packageDefinition = protoLoader.loadSync(PROTO_PATH, options);

const DeliveryService =
  grpc.loadPackageDefinition(packageDefinition).DeliveryService;

const client = new DeliveryService(
  "localhost:50051",
  grpc.credentials.createInsecure()
);

module.exports = client;
