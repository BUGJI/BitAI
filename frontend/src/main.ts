import { createApp } from 'vue';
import ArcoVue from '@arco-design/web-vue';
import ArcoVueIcon from '@arco-design/web-vue/es/icon';
import '@arco-design/web-vue/dist/arco.css';
import './styles/tokens.css';
import './styles/app.css';
import App from './App.vue';
import { router } from './app/router';
import { createPinia } from 'pinia';

createApp(App).use(createPinia()).use(router).use(ArcoVue).use(ArcoVueIcon).mount('#app');
