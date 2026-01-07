import { writable } from 'svelte/store';

export const activeTab = writable('dashboard');
export const stockCode = writable('005930');
export const stockName = writable('삼성전자');
