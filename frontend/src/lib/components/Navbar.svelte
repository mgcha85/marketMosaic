<script>
  import { createEventDispatcher } from "svelte";

  export let title = "Market Mosaic";
  export let selectedDate = "";
  export let stockCode = "";
  export let stockName = "";

  const dispatch = createEventDispatcher();

  let searchInput = "";

  function handleSearch() {
    if (searchInput.trim()) {
      dispatch("search", { code: searchInput.trim() });
      searchInput = "";
    }
  }

  function handleKeydown(e) {
    if (e.key === "Enter") {
      handleSearch();
    }
  }
</script>

<div class="navbar bg-base-100 shadow-lg border-b border-base-300">
  <div class="navbar-start">
    <a href="/" class="btn btn-ghost normal-case text-xl font-bold gap-2">
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="h-6 w-6"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
        />
      </svg>
      {title}
    </a>
  </div>

  <div class="navbar-center hidden lg:flex">
    <div class="stats shadow bg-base-200">
      <div class="stat py-2 px-4">
        <div class="stat-title text-xs">현재 종목</div>
        <div class="stat-value text-sm text-primary">
          {stockName} ({stockCode})
        </div>
      </div>
    </div>
  </div>

  <div class="navbar-end gap-2">
    <!-- Date Picker -->
    <div class="form-control">
      <label class="label py-0">
        <span class="label-text text-xs opacity-70">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-3 w-3 inline mr-1"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
            />
          </svg>
          기준일
        </span>
      </label>
      <input
        type="datetime-local"
        class="input input-bordered input-sm w-52 bg-base-200 focus:bg-base-100 transition-colors"
        bind:value={selectedDate}
      />
    </div>

    <!-- Search Stock -->
    <div class="form-control hidden md:block">
      <label class="label py-0">
        <span class="label-text text-xs opacity-70">종목 검색</span>
      </label>
      <div class="join">
        <input
          type="text"
          placeholder="종목코드 입력..."
          class="input input-bordered input-sm w-32 lg:w-40 join-item"
          bind:value={searchInput}
          on:keydown={handleKeydown}
        />
        <button
          class="btn btn-sm btn-primary join-item"
          on:click={handleSearch}
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-4 w-4"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
            />
          </svg>
        </button>
      </div>
    </div>

    <!-- User Avatar -->
    <div class="dropdown dropdown-end">
      <div tabindex="0" role="button" class="btn btn-ghost btn-circle avatar">
        <div
          class="w-10 rounded-full ring ring-primary ring-offset-base-100 ring-offset-2"
        >
          <img
            alt="User"
            src="https://daisyui.com/images/stock/photo-1534528741775-53994a69daeb.jpg"
          />
        </div>
      </div>
      <ul
        tabindex="0"
        class="mt-3 z-[1] p-2 shadow menu menu-sm dropdown-content bg-base-100 rounded-box w-52"
      >
        <li>
          <a class="justify-between">
            Profile
            <span class="badge">New</span>
          </a>
        </li>
        <li><a>Settings</a></li>
        <li><a>Logout</a></li>
      </ul>
    </div>
  </div>
</div>
