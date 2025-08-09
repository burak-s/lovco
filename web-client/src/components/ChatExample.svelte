<script>
  import { chatClient, handleGrpcError } from '../lib/grpc-client.js';
  import { 
    JoinChatRequest, 
    ChatMessageRequest, 
    EndChatRequest,
  } from '../generated/chat/chat_pb';
  import { onMount } from 'svelte';
  
  let leftoverId = '';
  let userId = '';
  let messageText = '';
  let messages = [];
  let chatStream = null;
  let isConnected = false;

  // Add queue state variables
  let queueStatus = {
    position: -1,
    queuedCount: 0,
    isQueued: false
  };
  let queueStream = null;

  function joinChat() {
    if (!leftoverId || !userId) return;

    const request = new JoinChatRequest();
    request.setLeftoverId(leftoverId);
    request.setUserId(userId);

    try {
      chatStream = chatClient.joinChat(request);
      isConnected = true;
      
      chatStream.on('data', (message) => {
        messages = [...messages, {
          id: message.getId(),
          leftoverId: message.getLeftoverId(),
          userId: message.getUserId(),
          message: message.getMessage(),
          image: message.getImage(),
          createdAt: message.getCreatedAt()
        }];
      });

      chatStream.on('error', (error) => {
        console.error('Chat stream error:', handleGrpcError(error));
        isConnected = false;
      });

      chatStream.on('end', () => {
        console.log('Chat stream ended');
        isConnected = false;
      });

    } catch (error) {
      console.error('Failed to join chat:', handleGrpcError(error));
    }
  }

  function sendMessage() {
    if (!messageText || !isConnected) return;

    const request = new ChatMessageRequest();
    request.setLeftoverId(leftoverId);
    request.setUserId(userId);
    request.setMessage(messageText);

    chatClient.sendMessage(request, {}, (error, response) => {
      if (error) {
        console.error('Failed to send message:', handleGrpcError(error));
      } else {
        messageText = '';
      }
    });
  }

  function endChat() {
    if (chatStream) {
      chatStream.cancel();
      chatStream = null;
    }
    
    // Also cancel queue stream
    if (queueStream) {
      queueStream.cancel();
      queueStream = null;
    }

    const request = new EndChatRequest();
    request.setLeftoverId(leftoverId);
    request.setUserId(userId);

    chatClient.endChatSession(request, {}, (error, response) => {
      if (error) {
        console.error('Failed to end chat:', handleGrpcError(error));
      } else {
        messages = [];
        isConnected = false;
      }
    });
  }

  function watchQueue(){
    const request = new JoinChatRequest();
    request.setLeftoverId(leftoverId);
    request.setUserId(userId);

    try {
      queueStream = chatClient.watchChatQueue(request);
      
      queueStream.on('data', (message) => {
        // Handle queue status updates
        queueStatus = {
          position: message.getPosition(),
          queuedCount: message.getQueuedCount(),
          isQueued: message.getPosition() > 0
        };
        console.log('Queue status:', queueStatus);
      });

      queueStream.on('error', (error) => {
        console.error('Queue error:', handleGrpcError(error));
      });

      queueStream.on('end', () => {
        console.log('Queue stream ended');
      });

    } catch (error) {
      console.error('Failed to watch queue:', handleGrpcError(error));
    }
  }

  onMount(() => {
    leftoverId = window.location.hash.split('?')[1].split('=')[1];
    userId = sessionStorage.getItem('userId');

    watchQueue();
  });
</script>

<div class="max-w-2xl mx-auto p-6 bg-gray-50 min-h-screen">
  <h2 class="text-3xl font-bold text-gray-800 mb-8 text-center">Chat Service</h2>
  
  <!-- Queue Status Display -->
  {#if queueStatus.isQueued}
    <div class="bg-yellow-100 border border-yellow-400 text-yellow-700 px-4 py-3 rounded mb-4">
      <div class="flex items-center">
        <svg class="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z" clip-rule="evenodd"></path>
        </svg>
        <div>
          <strong>You are in queue</strong>
          <p class="text-sm">Position: {queueStatus.position} of {queueStatus.queuedCount} people waiting</p>
        </div>
      </div>
    </div>
  {/if}

  {#if !isConnected && !queueStatus.isQueued}
  <div class="bg-white rounded-lg shadow-md p-6 mb-6">
    <h3 class="text-lg font-semibold text-gray-700 mb-4">
      Ask leftover owner for item details and delivery details
    </h3>
    <button 
      on:click={joinChat}
      disabled={!leftoverId || !userId}
      class="w-full bg-green-600 hover:bg-green-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white font-medium py-2 px-4 rounded-md transition duration-200"
    >
      Join Chat
    </button>
  </div>
    <div class="bg-white rounded-lg shadow-md p-8 text-center">
      <div class="w-16 h-16 bg-gray-200 rounded-full mx-auto mb-4 flex items-center justify-center">
        <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"></path>
        </svg>
      </div>
      <h3 class="text-lg font-medium text-gray-700 mb-2">Ready to Chat</h3>
      <p class="text-gray-500">Enter your details above to join the chat room</p>
    </div>
  {/if}
    
    {#if isConnected && !queueStatus.isQueued}
      <div class="mt-3 flex items-center flex-col text-sm text-green-600">
        <button 
          on:click={endChat}
          class="w-full bg-red-600 hover:bg-red-700 text-white font-medium py-2 px-4 rounded-md transition duration-200"
        >
          End Chat
        </button>
        <div class="w-2 h-2 bg-green-500 rounded-full mr-2 animate-pulse"></div>
        Connected to chat for leftover: {leftoverId}
      </div>
    {/if}


  {#if isConnected && !queueStatus.isQueued}
    <div class="bg-white rounded-lg shadow-md mb-4 flex flex-col h-96">
      <div class="p-4 border-b border-gray-200">
        <h3 class="text-lg font-semibold text-gray-700">Messages</h3>
      </div>
      
      <div class="flex-1 overflow-y-auto p-4 space-y-3">
        {#each messages as message}
          <div class="flex flex-col space-y-2">
            <div class="flex items-center space-x-2">
              <div class="w-8 h-8 bg-blue-500 text-white rounded-full flex items-center justify-center text-sm font-medium">
                {message.userId.slice(0, 2).toUpperCase()}
              </div>
              <div class="flex-1">
                <div class="bg-gray-100 rounded-lg p-3">
                  <div class="flex items-center justify-between mb-1">
                    <span class="text-sm font-medium text-gray-700">User {message.userId}</span>
                    {#if message.createdAt}
                      <span class="text-xs text-gray-500">{new Date(message.createdAt)}</span>
                    {/if}
                  </div>
                  <p class="text-gray-800">{message.message}</p>
                  {#if message.image}
                    <img 
                      src={message.image} 
                      alt="Message attachment" 
                      class="mt-2 max-w-full h-32 object-cover rounded-md border border-gray-200"
                    />
                  {/if}
                </div>
              </div>
            </div>
          </div>
        {/each}
        
        {#if messages.length === 0}
          <div class="text-center py-8">
            <p class="text-gray-500">No messages yet</p>
            <p class="text-gray-400 text-sm mt-1">Start the conversation!</p>
          </div>
        {/if}
      </div>
    </div>

    <!-- Message Input -->
    <div class="bg-white rounded-lg shadow-md p-4">
      <div class="flex space-x-3">
        <input 
          bind:value={messageText} 
          placeholder="Type your message..." 
          on:keypress={(e) => e.key === 'Enter' && sendMessage()}
          class="flex-1 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
        />
        <button 
          on:click={sendMessage}
          disabled={!messageText.trim()}
          class="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed text-white font-medium py-2 px-6 rounded-md transition duration-200"
        >
          Send
        </button>
      </div>
      <p class="text-xs text-gray-500 mt-2">Press Enter to send</p>
    </div>
  {/if}
</div>