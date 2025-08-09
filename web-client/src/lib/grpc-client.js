import { ChatServiceClient } from '../generated/chat/chat_grpc_web_pb';
import { LeftoverServiceClient } from '../generated/leftover/leftover_grpc_web_pb';

const GRPC_WEB_URL = 'http://localhost:8080';

export const chatClient = new ChatServiceClient(GRPC_WEB_URL);
export const leftoverClient = new LeftoverServiceClient(GRPC_WEB_URL);

export function handleGrpcError(error) {
  return {
    code: error.code || 'UNKNOWN',
    message: error.message || 'An unknown error occurred'
  };
}

export function createMetadata(headers = {}) {
  const metadata = {};
  Object.keys(headers).forEach(key => {
    metadata[key] = headers[key];
  });
  return metadata;
}
