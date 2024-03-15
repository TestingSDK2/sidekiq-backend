// test.js
// get all news
const client = require("./client");

console.log({client})

const notification = {
    notificationId: '65c9cf413aa825a43e75099e',
    recipientProfileId: '321',
    senderProfileId: '283',
    thingType: 'CONNECTION',
    thingId: '65c9cf36918a090b80fa4214',
    isRead: true,
    actionType: 'AcceptConnectionRequest',
    notificationText: 'WEB test WEB has accepted your request',
    createdDate: {
        seconds: 1644677809,
        nanos: 599000000
    }
};

client.DeliverNotification(notification, (err, response) => {
    if (err) {
        console.error('Error:', err);
        return;
    }
    console.log('Response:', response);
});

// add a news
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