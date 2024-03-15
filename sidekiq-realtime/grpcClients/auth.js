const path = require("path");
const grpc = require("@grpc/grpc-js");
var protoLoader = require("@grpc/proto-loader");
const { AuthGrpcClientUrl } = require("../constants/env");
const PROTO_PATH = path.resolve(
  __dirname,
  "../../sidekiq-auth-server/protobuf/v1/auth.proto"
);

const options = {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
};

var packageDefinition = protoLoader.loadSync(PROTO_PATH, options);

const authProto = grpc.loadPackageDefinition(packageDefinition);

const authService = authProto.auth.v1.AuthService;

const authGrpcClient = new authService(
  AuthGrpcClientUrl,
  grpc.credentials.createInsecure()
);

const validateJWT = async (token, profileId, shouldValidateProfile = true) => {
  try {
    return new Promise((resolve, reject) => {
      authGrpcClient.ValidateUser(
        {
          token,
          profileID: profileId,
          isProfileValidate: shouldValidateProfile,
        },
        (err, response) => {
          if (err) {
            console.error("Error validating JWT:", err);
            reject(err);
          } else {
            resolve(response.data);
          }
        }
      );
    });
  } catch (error) {
    return null;
  }
};

module.exports = { validateJWT };
