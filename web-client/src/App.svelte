<script>
  import { onMount } from 'svelte';
  import ChatExample from '@components/ChatExample.svelte';
  import LeftoverExample from '@components/LeftoverExample.svelte';
  import NotFound from '@components/NotFound.svelte';
  import { v4 as uuidv4 } from 'uuid';

  let route = '/leftover';

  function setRouteFromHash() {
    const h = (window.location.hash || '#/leftover').slice(1);
    route = h || '/leftover';
  }

  onMount(() => {
    setRouteFromHash();
    window.addEventListener('hashchange', setRouteFromHash);
    if (!sessionStorage.getItem('userId')) {
      sessionStorage.setItem('userId', uuidv4());
    }
    return () => window.removeEventListener('hashchange', setRouteFromHash);
  });
</script>

<main class="flex flex-col items-center justify-center h-screen">
  {#if route === '/leftover'}
    <LeftoverExample />
  {:else if route.startsWith('/chat')}
    <ChatExample />
  {:else}
    <NotFound path={route} />
  {/if}
</main>