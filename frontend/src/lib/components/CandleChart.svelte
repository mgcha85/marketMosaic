<script>
    import { onMount, onDestroy, afterUpdate } from "svelte";
    import { createChart, ColorType } from "lightweight-charts";

    export let data = []; // { time, open, high, low, close }
    export let height = 350;

    let chartContainer;
    let chart;
    let candleSeries;

    onMount(() => {
        if (!chartContainer) return;

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
            },
        });

        // Korean Style: Up = Red, Down = Blue
        candleSeries = chart.addCandlestickSeries({
            upColor: "#EF5350", // Red for Up
            downColor: "#2962FF", // Blue for Down
            borderVisible: false,
            wickUpColor: "#EF5350",
            wickDownColor: "#2962FF",
        });

        if (data && data.length > 0) {
            candleSeries.setData(data);
        }

        const handleResize = () => {
            if (chartContainer) {
                chart.applyOptions({ width: chartContainer.clientWidth });
            }
        };

        window.addEventListener("resize", handleResize);

        return () => {
            window.removeEventListener("resize", handleResize);
            chart.remove();
        };
    });

    // React to data changes
    $: if (candleSeries && data) {
        candleSeries.setData(data);
    }
</script>

<div
    bind:this={chartContainer}
    class="w-full relative"
    style="height: {height}px"
/>
