import { ChatServiceClient } from '../generated/chat/chat_grpc_web_pb';
import { LeftoverServiceClient } from '../generated/leftover/leftover_grpc_web_pb';

// Configure the gRPC-Web clients
const GRPC_WEB_URL = 'http://localhost:8080'; // Envoy proxy URL

// Create client instances
export const chatClient = new ChatServiceClient(GRPC_WEB_URL);
export const leftoverClient = new LeftoverServiceClient(GRPC_WEB_URL);

// Helper function to handle gRPC errors
export function handleGrpcError(error) {
  console.error('gRPC Error:', error);
  return {
    code: error.code || 'UNKNOWN',
    message: error.message || 'An unknown error occurred'
  };
}

// Helper function to create metadata
export function createMetadata(headers = {}) {
  const metadata = {};
  Object.keys(headers).forEach(key => {
    metadata[key] = headers[key];
  });
  return metadata;
}
