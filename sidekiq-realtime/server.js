require("dotenv").config();
const grpc = require("@grpc/grpc-js");
const PROTO_PATH = "./protobuf/v1/delivery.proto";
var protoLoader = require("@grpc/proto-loader");
const { createWebSocketServer, emitToSpecificClients } = require("./socket");
require("./redis");

const options = {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
};

const startUp = async () => {
  var packageDefinition = protoLoader.loadSync(PROTO_PATH, options);
  const deliveryProto = grpc.loadPackageDefinition(packageDefinition);

  const server = new grpc.Server();
  const io = await createWebSocketServer();

  server.addService(deliveryProto.DeliveryService.service, {
    DeliverNotification: (call, callback) => {
      try {
        const notification = call.request;
        let receiptIds = [];
        if (notification?.recipientProfileId) {
          receiptIds.push(notification.recipientProfileId);
        }
        if (notification?.receiptProfileIds) {
          receiptIds = [...receiptIds, notification.receiptProfileIds];
        }
        console.log({ receiptIds });
        emitToSpecificClients("notification", notification, receiptIds);
        callback(null, { acknowledgment: "DELIVERED" });
      } catch (error) {
        console.log({ error });
        callback(null, { acknowledgment: "UNABLE_TO_DELIVER" });
      }
    },
    DeliverMessage: (call, callback) => {
      try {
        const message = call.request;
        emitToSpecificClients(
          "message",
          message,
          message?.receiptProfileIds || []
        );
        callback(null, { acknowledgment: "DELIVERED" });
      } catch (error) {
        callback(null, { acknowledgment: "UNABLE_TO_DELIVER" });
      }
    },
  });

  server.bindAsync(
    process.env.GRPC_BIND,
    grpc.ServerCredentials.createInsecure(),
    (error, port) => {
      console.log("Server at port:", port);
      console.log("Server running at http://127.0.0.1:", port);
      server.start();
    }
  );
};

startUp();
