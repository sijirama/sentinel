import { defineConfig } from 'astro/config';
import tailwind from '@astrojs/tailwind';

import react from '@astrojs/react';

export default defineConfig({
    integrations: [tailwind(), react()],
    devToolbar: {
        enabled: false
    },
    vite: {
        server: {
            proxy: {
                '/api': {
                    target: 'http://backend:8080',
                    changeOrigin: true,
                    rewrite: (path) => path.replace(/^\/api/, '')
                }
            }
        }
    }
});
