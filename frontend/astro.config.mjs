import { defineConfig } from 'astro/config';
import tailwind from '@astrojs/tailwind';

import react from '@astrojs/react';

// https://astro.build/config
export default defineConfig({
    integrations: [tailwind(), react()],
    // vite: {
    //     server: {
    //         proxy: {
    //             '/status': {
    //                 target: 'http://localhost:8080/status',
    //                 changeOrigin: true,
    //                 rewrite: (path) => path.replace(/^\/api/, '')
    //             }
    //         }
    //     }
    // }
});
