// test.js
// get all news
const client = require("./client");

console.log({ client });

const notification = {
  notificationId: "65c9cf413aa825a43e75099e",
  receiptProfileIds: ["283"],
  senderProfileId: "283",
  thingType: "CONNECTION",
  thingId: "65c9cf36918a090b80fa4214",
  isRead: true,
  actionType: "AcceptConnectionRequest",
  notificationText: "WEB test WEB has accepted your request",
  createdDate: {
    seconds: 1644677809,
    nanos: 599000000,
  },
};

client.DeliverNotification(notification, (err, response) => {
  if (err) {
    console.error("Error:", err);
    return;
  }
  console.log("Response:", response);
});

// client.addNews(
//   {
//     title: "Title news 3",
//     body: "Body content 3",
//     postImage: "Image URL here",
//   },
//   (error, news) => {
//     if (error) throw error;
//     console.log("Successfully created a news.");
//   }
// );

// // edit a news
// client.editNews(
//   {
//     id: 2,
//     body: "Body content 2 edited.",
//     postImage: "Image URL edited.",
//     title: "Title for 2 edited.",
//   },
//   (error, news) => {
//     if (error) throw error;
//     console.log("Successfully edited a news.");
//   }
// );

// // delete a news
// client.deleteNews(
//   {
//     id: 2,
//   },
//   (error, news) => {
//     if (error) throw error;
//     console.log("Successfully deleted a news item.");
//   }
// );

// const wss = new WebSocket.Server({ port: WebSocketPort });

// wss.on("listening", () => {
//   console.log("Web socket server starting on ", WebSocketPort);
// });

// wss.on("connection", async function connection(ws, req) {
//   const profileID = req.headers.profileid;
//   const connectionId = req.headers["sec-websocket-key"];

//   // check if token is authorized
//   const user = await authenticateUser(req);
//   if (!user) {
//     // close socket if user not authorized
//     ws.close();
//     return;
//   }
//   await addConnectionId(profileID, connectionId);

//   // report new connection to redis
//   ws.on("close", async function close(ws) {
//     await deleteConnection(profileID, connectionId);
//     console.log("Client disconnected");
//   });
// });

// return wss;
// Initialize Express app
