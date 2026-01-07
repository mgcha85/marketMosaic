<script>
    import { onMount } from "svelte";
    import {
        createChart,
        ColorType,
        CandlestickSeries,
    } from "lightweight-charts";

    export let data = []; // { time, open, high, low, close }
    export let height = 350;
    export let onVisibleRangeChange = null; // Callback for lazy loading

    let chartContainer;
    let chart = null;
    let candleSeries = null;

    onMount(() => {
        console.log("CandleChart: onMount, data length:", data?.length);

        if (!chartContainer) {
            console.error("CandleChart: chartContainer is null");
            return;
        }

        try {
            chart = createChart(chartContainer, {
                layout: {
                    background: { type: ColorType.Solid, color: "white" },
                    textColor: "black",
                },
                width: chartContainer.clientWidth,
                height: height,
                grid: {
                    vertLines: { color: "#f0f0f0" },
                    horzLines: { color: "#f0f0f0" },
                },
                rightPriceScale: {
                    borderColor: "#d1d4dc",
                },
                timeScale: {
                    borderColor: "#d1d4dc",
                    timeVisible: true,
                    secondsVisible: false,
                },
            });
            console.log("CandleChart: createChart SUCCESS");
        } catch (e) {
            console.error("CandleChart: createChart FAILED:", e);
            return;
        }

        try {
            candleSeries = chart.addSeries(CandlestickSeries, {
                upColor: "#EF5350",
                downColor: "#2962FF",
                borderVisible: false,
                wickUpColor: "#EF5350",
                wickDownColor: "#2962FF",
            });
            console.log("CandleChart: addSeries SUCCESS");
        } catch (e) {
            console.error("CandleChart: addSeries FAILED:", e);
            return;
        }

        // Subscribe to visible time range changes for lazy loading
        if (onVisibleRangeChange) {
            chart.timeScale().subscribeVisibleTimeRangeChange((range) => {
                onVisibleRangeChange(range);
            });
        }

        // Set initial data
        if (data && data.length > 0) {
            console.log("CandleChart: setting", data.length, "initial candles");
            try {
                candleSeries.setData(data);
                chart.timeScale().fitContent();
                console.log("CandleChart: setData SUCCESS");
            } catch (e) {
                console.error("CandleChart: setData FAILED:", e);
            }
        }

        const handleResize = () => {
            if (chartContainer && chart) {
                chart.applyOptions({ width: chartContainer.clientWidth });
            }
        };
        window.addEventListener("resize", handleResize);

        return () => {
            window.removeEventListener("resize", handleResize);
            if (chart) chart.remove();
        };
    });

    // Reactive update when data changes - don't call fitContent to preserve scroll position
    $: if (chart && candleSeries && data && data.length > 0) {
        console.log("CandleChart REACTIVE: updating", data.length, "candles");
        try {
            candleSeries.setData(data);
            // Don't call fitContent here - preserve scroll position when loading more data
        } catch (e) {
            console.error("CandleChart REACTIVE error:", e);
        }
    }
</script>

<div
    bind:this={chartContainer}
    class="w-full relative"
    style="height: {height}px"
/>
