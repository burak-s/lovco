<script>
  import { leftoverClient, handleGrpcError } from '../lib/grpc-client.js';
  import { 
    LeftoverRequest, 
    LeftoverIdentity, 
    Leftover,
    DeleteRequest 
  } from '../generated/leftover/leftover_pb';
  
  let leftovers = [];
  let newLeftover = {
    ownerId: '',
    name: '',
    description: '',
    image: '',
    longitude: 0,
    latitude: 0
  };
  let selectedLeftoverId = '';

  function addLeftover() {
    const request = new LeftoverRequest();
    request.setOwnerId(newLeftover.ownerId);
    request.setName(newLeftover.name);
    request.setDescription(newLeftover.description);
    request.setImage(newLeftover.image);
    request.setLongitude(newLeftover.longitude);
    request.setLatitude(newLeftover.latitude);

    leftoverClient.addLeftover(request, {}, (error, response) => {
      if (error) {
        console.error('Failed to add leftover:', handleGrpcError(error));
      } else {
        console.log('Leftover added successfully');
        newLeftover = {
          ownerId: sessionStorage.getItem('userId'),
          name: '',
          description: '',
          image: '',
          longitude: 0,
          latitude: 0
        };
        getLeftovers();
      }
    });
  }

  function getLeftovers() {
    const request = new LeftoverRequest();

    leftoverClient.getLeftovers(request, {}, (error, response) => {
      if (error) {
        console.error('Failed to get leftovers:', handleGrpcError(error));
      } else {
        leftovers = response.getItemsList().map(leftover => ({
          id: leftover.getId(),
          ownerId: leftover.getOwnerId(),
          name: leftover.getName(),
          description: leftover.getDescription(),
          image: leftover.getImage(),
          longitude: leftover.getLongitude(),
          latitude: leftover.getLatitude()
        }));
      }
    });
  }

  function getLeftover(id) {
    const request = new LeftoverIdentity();
    request.setId(id);

    leftoverClient.getLeftover(request, {}, (error, response) => {
      if (error) {
        console.error('Failed to get leftover:', handleGrpcError(error));
      } else {
        console.log('Leftover details:', {
          id: response.getId(),
          ownerId: response.getOwnerId(),
          name: response.getName(),
          description: response.getDescription(),
          image: response.getImage(),
          longitude: response.getLongitude(),
          latitude: response.getLatitude()
        });
      }
    });
  }

  function deleteLeftover(id, ownerId) {
    const request = new DeleteRequest();
    request.setId(id);
    request.setOwnerId(ownerId);

    leftoverClient.deleteLeftover(request, {}, (error, response) => {
      if (error) {
        console.error('Failed to delete leftover:', handleGrpcError(error));
      } else {
        console.log('Leftover deleted successfully');
        getLeftovers(); // Refresh the list
      }
    });
  }

  getLeftovers();
</script>

<div class="max-w-4xl mx-auto p-6 bg-gray-50 min-h-screen">
  <h2 class="text-3xl font-bold text-gray-800 mb-8 text-center">Leftover Service</h2>
  
  <div class="bg-white rounded-lg shadow-md p-6 mb-8">
    <h3 class="text-xl font-semibold text-gray-700 mb-4">New Leftover</h3>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
      <input 
        bind:value={newLeftover.name} 
        placeholder="Name" 
        class="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent col-span-2"
      />
      <textarea 
        bind:value={newLeftover.description} 
        placeholder="Description" 
        class="px-3 py-2 border h-30 resize-none border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent md:col-span-2 align-top"
        style="vertical-align: top; text-align: start;"
      ></textarea>
      <!-- TODO: Add cloud storage -->
      <input 
        bind:value={newLeftover.image} 
        placeholder="Image" 
        class="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent md:col-span-2"
      />
      <input 
        bind:value={newLeftover.longitude} 
        type="number" 
        step="0.000001" 
        placeholder="Longitude" 
        class="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
      />
      <input 
        bind:value={newLeftover.latitude} 
        type="number" 
        step="0.000001" 
        placeholder="Latitude" 
        class="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
      />
    </div>
    <button 
      on:click={addLeftover}
      class="w-full bg-blue-600 hover:bg-blue-700 text-white font-medium py-2 px-4 rounded-md transition duration-200 ease-in-out transform hover:scale-105"
    >
      Add Leftover
    </button>
  </div>

  <div class="bg-white rounded-lg shadow-md p-6">
    <div class="flex justify-between items-center mb-6">
      <h3 class="text-xl font-semibold text-gray-700">Leftovers</h3>
      <button 
        on:click={getLeftovers}
        class="bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-4 rounded-md transition duration-200 ease-in-out"
      >
        Refresh List
      </button>
    </div>
    
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
      {#each leftovers as leftover}
      <a href={`#/chat?lid=${leftover.id}`}>
        <div class="bg-gray-50 border border-gray-200 rounded-lg p-4 hover:shadow-lg transition duration-200">
          <h4 class="text-lg font-semibold text-gray-800 mb-3">{leftover.name}</h4>
          
          <div class="space-y-2 mb-4">
            <p class="text-sm text-gray-600">
              <span class="font-medium text-gray-700">Description:</span> {leftover.description}
            </p>
            <p class="text-sm text-gray-600">
              <span class="font-medium text-gray-700">Location:</span> 
              <span class="font-mono text-xs">{leftover.latitude}, {leftover.longitude}</span>
            </p>
          </div>
          
          {#if leftover.image}
            <img 
              src={leftover.image} 
              alt={leftover.name} 
              class="w-full h-32 object-cover rounded-md mb-4"
            />
          {/if}
          
          <div class="flex space-x-2">
            <button 
              on:click={() => getLeftover(leftover.id)}
              class="flex-1 bg-gray-600 hover:bg-gray-700 text-white text-sm font-medium py-2 px-3 rounded-md transition duration-200"
            >
              Demand or Get Details
            </button>
            {#if leftover.ownerId === sessionStorage.getItem('userId')}
            <button 
              on:click={() => deleteLeftover(leftover.id, sessionStorage.getItem('userId'))}
              class="flex-1 bg-red-600 hover:bg-red-700 text-white text-sm font-medium py-2 px-3 rounded-md transition duration-200"
            >
              Delete
            </button>
            {/if}
          </div>
        </div>
      </a>
      {/each}
    </div>
    
    {#if leftovers.length === 0}
      <div class="text-center py-8">
        <p class="text-gray-500 text-lg">No leftovers found</p>
        <p class="text-gray-400 text-sm mt-2">Add a new leftover to get started</p>
      </div>
    {/if}
  </div>
</div>