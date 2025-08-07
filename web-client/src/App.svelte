<script>
  import ChatExample from './components/ChatExample.svelte';
  import LeftoverExample from './components/LeftoverExample.svelte';
  
  let activeTab = 'leftover';
</script>

<main>
  <h1>gRPC-Web Demo</h1>

  <div class="tabs">
    <button 
      class:active={activeTab === 'leftover'} 
      on:click={() => activeTab = 'leftover'}
    >
      Leftover Service
    </button>
    <button 
      class:active={activeTab === 'chat'} 
      on:click={() => activeTab = 'chat'}
    >
      Chat Service
    </button>
  </div>

  {#if activeTab === 'leftover'}
    <LeftoverExample />
  {:else if activeTab === 'chat'}
    <ChatExample />
  {/if}
</main>

<style>
  main {
    padding: 20px;
    font-family: Arial, sans-serif;
  }
  
  h1 {
    text-align: center;
    color: #333;
  }
  
  .tabs {
    display: flex;
    justify-content: center;
    margin-bottom: 20px;
    gap: 10px;
  }
  
  .tabs button {
    padding: 10px 20px;
    border: 1px solid #ddd;
    border-radius: 4px;
    background-color: white;
    cursor: pointer;
    transition: background-color 0.2s;
  }
  
  .tabs button:hover {
    background-color: #f0f0f0;
  }
  
  .tabs button.active {
    background-color: #007bff;
    color: white;
    border-color: #007bff;
  }
</style>