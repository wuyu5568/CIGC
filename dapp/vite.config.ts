import { join } from "path";
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import postcssPluginPx2rem from "postcss-plugin-px2rem";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
    plugins: [vue({
        reactivityTransform: true,
    }),],
    resolve: {
        alias: {
            '@': join(__dirname, "src"),
        }
    },
    server: {
        host: true,
        open: true,
        port: 85,
        proxy: {
          '/api': mode === 'cigc' ? {
            target: process.env.VITE_PROXY || 'http://127.0.0.1:8000',
            changeOrigin: true,
          } : {
            target: 'http://44.192.48.164',
            changeOrigin: true,
            rewrite: (path) => path.replace(/^\/api/, '')
          }
        }
    },
    build: {
        outDir: "dist",
        assetsDir: "static",
        assetsInlineLimit: 150000
    },
    css: {
        preprocessorOptions: {
            less: {
                charset: false,
                additionalData: '@import "./src/style/global.less";',
            }
        },
        postcss: {
            plugins: [
                postcssPluginPx2rem({
                    rootValue: 37.5,
                    exclude: /(node_module)/,
                    mediaQuery: false,
                }),
            ]
        },
    }
}))
