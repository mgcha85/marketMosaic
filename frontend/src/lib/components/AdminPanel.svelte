<script>
    import { onMount } from "svelte";

    let status = null;
    let config = {};
    let isLoading = true;
    let saveMessage = "";

    async function fetchData() {
        try {
            const [statusRes, configRes] = await Promise.all([
                fetch("/admin/status"),
                fetch("/admin/config"),
            ]);

            if (statusRes.ok) status = (await statusRes.json()).status;
            if (configRes.ok) config = await configRes.json();
        } catch (e) {
            console.error(e);
        } finally {
            isLoading = false;
        }
    }

    async function saveConfig() {
        try {
            const res = await fetch("/admin/config", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(config),
            });
            if (res.ok) {
                saveMessage = "Saved successfully! Restart backend to apply.";
                setTimeout(() => (saveMessage = ""), 5000);
            } else {
                saveMessage = "Failed to save.";
            }
        } catch (e) {
            saveMessage = "Error saving config";
        }
    }

    onMount(fetchData);
</script>

<div class="container mx-auto p-6 max-w-6xl">
    <div class="flex justify-between items-center mb-8">
        <h1 class="text-3xl font-bold">System Administration</h1>
        <span class="badge badge-accent badge-lg">PROD</span>
    </div>

    <!-- Status Section -->
    <div class="card bg-base-100 shadow-xl mb-8 border border-base-200">
        <div class="card-body">
            <h2 class="card-title mb-4">Data Ingestion Status</h2>
            <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                <!-- Judal Status -->
                <div class="stats shadow bg-base-200">
                    <div class="stat">
                        <div class="stat-figure text-primary">
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                fill="none"
                                viewBox="0 0 24 24"
                                class="inline-block w-8 h-8 stroke-current"
                                ><path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M13 10V3L4 14h7v7l9-11h-7z"
                                ></path></svg
                            >
                        </div>
                        <div class="stat-title">Themes (Judal)</div>
                        <div class="stat-value text-primary text-2xl">
                            {status?.judal?.since_last_update || "-"}
                        </div>
                        <div class="stat-desc">
                            Last: {status?.judal?.latest_log?.created_at?.slice(
                                0,
                                16,
                            ) || "-"}
                        </div>
                        <div class="stat-desc">
                            Stocks: {status?.judal?.latest_log?.stocks_count ||
                                0}
                        </div>
                    </div>
                </div>

                <!-- DART Status -->
                <div class="stats shadow bg-base-200">
                    <div class="stat">
                        <div class="stat-figure text-secondary">
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                fill="none"
                                viewBox="0 0 24 24"
                                class="inline-block w-8 h-8 stroke-current"
                                ><path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                                ></path></svg
                            >
                        </div>
                        <div class="stat-title">DART Data</div>
                        <div class="stat-value text-secondary text-2xl">
                            {status?.dart?.since_last_update || "-"}
                        </div>
                        <div class="stat-desc">
                            Last Filing: {status?.dart?.last_filing_date || "-"}
                        </div>
                        <div class="stat-desc">
                            {status?.dart?.days_ago ?? 0} days ago
                        </div>
                    </div>
                </div>

                <!-- News Status -->
                <div class="stats shadow bg-base-200">
                    <div class="stat">
                        <div class="stat-figure text-accent">
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                fill="none"
                                viewBox="0 0 24 24"
                                class="inline-block w-8 h-8 stroke-current"
                                ><path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M19 20H5a2 2 0 01-2-2V6a2 2 0 012-2h10a2 2 0 012 2v1m2 13a2 2 0 01-2-2V7m2 13a2 2 0 002-2V9a2 2 0 00-2-2h-2m-4-3H9M7 16h6M7 8h6v4H7V8z"
                                ></path></svg
                            >
                        </div>
                        <div class="stat-title">News Stream</div>
                        <div class="stat-value text-accent text-2xl">
                            Active
                        </div>
                        <div class="stat-desc">Real-time Ingestion</div>
                        <div class="stat-desc">
                            Cron: {config.news_fetch_cron || "Every 15m"}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Config Section -->
    <div class="card bg-base-100 shadow-xl border border-base-200">
        <div class="card-body">
            <h2 class="card-title mb-6">API Configuration (Overwrites .env)</h2>

            {#if isLoading}
                <div class="flex justify-center">
                    <span class="loading loading-spinner"></span>
                </div>
            {:else}
                <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-4">
                    <div
                        class="divider col-span-2 text-xs opacity-50 font-bold"
                    >
                        Data Sources
                    </div>

                    <div class="form-control w-full">
                        <label class="label"
                            ><span class="label-text">DART API Key</span></label
                        >
                        <input
                            type="text"
                            bind:value={config.dart_api_key}
                            class="input input-bordered w-full font-mono text-sm"
                        />
                    </div>
                    <div class="hidden md:block"></div>

                    <div class="form-control w-full">
                        <label class="label"
                            ><span class="label-text">Naver Client ID</span
                            ></label
                        >
                        <input
                            type="text"
                            bind:value={config.naver_client_id}
                            class="input input-bordered w-full font-mono text-sm"
                        />
                    </div>
                    <div class="form-control w-full">
                        <label class="label"
                            ><span class="label-text">Naver Client Secret</span
                            ></label
                        >
                        <input
                            type="password"
                            bind:value={config.naver_client_secret}
                            class="input input-bordered w-full font-mono text-sm"
                        />
                    </div>

                    <div class="form-control w-full">
                        <label class="label"
                            ><span class="label-text"
                                >News Fetch Schedule (Cron)</span
                            ></label
                        >
                        <input
                            type="text"
                            bind:value={config.news_fetch_cron}
                            class="input input-bordered w-full font-mono text-sm"
                            placeholder="*/15 * * * * or @hourly"
                        />
                    </div>
                    <div class="hidden md:block"></div>

                    <div
                        class="divider col-span-2 text-xs opacity-50 font-bold"
                    >
                        Trading / Brokerage
                    </div>

                    <div class="form-control w-full">
                        <label class="label"
                            ><span class="label-text">Kiwoom App Key</span
                            ></label
                        >
                        <input
                            type="text"
                            bind:value={config.kiwoom_app_key}
                            class="input input-bordered w-full font-mono text-sm"
                        />
                    </div>
                    <div class="form-control w-full">
                        <label class="label"
                            ><span class="label-text">Kiwoom Secret</span
                            ></label
                        >
                        <input
                            type="password"
                            bind:value={config.kiwoom_app_secret}
                            class="input input-bordered w-full font-mono text-sm"
                        />
                    </div>

                    <div class="form-control w-full">
                        <label class="label"
                            ><span class="label-text">Alpaca Key</span></label
                        >
                        <input
                            type="text"
                            bind:value={config.alpaca_api_key}
                            class="input input-bordered w-full font-mono text-sm"
                        />
                    </div>
                    <div class="form-control w-full">
                        <label class="label"
                            ><span class="label-text">Alpaca Secret</span
                            ></label
                        >
                        <input
                            type="password"
                            bind:value={config.alpaca_api_secret}
                            class="input input-bordered w-full font-mono text-sm"
                        />
                    </div>
                </div>

                <div class="card-actions justify-end mt-8">
                    {#if saveMessage}
                        <div
                            class="alert alert-success py-2 px-4 w-auto inline-flex"
                        >
                            <svg
                                xmlns="http://www.w3.org/2000/svg"
                                class="stroke-current shrink-0 h-6 w-6"
                                fill="none"
                                viewBox="0 0 24 24"
                                ><path
                                    stroke-linecap="round"
                                    stroke-linejoin="round"
                                    stroke-width="2"
                                    d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                                /></svg
                            >
                            <span>{saveMessage}</span>
                        </div>
                    {/if}
                    <button
                        class="btn btn-primary btn-wide"
                        on:click={saveConfig}>Save Changes</button
                    >
                </div>
            {/if}
        </div>
    </div>
</div>
