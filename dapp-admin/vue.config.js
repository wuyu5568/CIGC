const path = require('path')
const webpack = require('webpack')

function resolve (dir) {
    return path.join(__dirname, dir)
}

// vue.config.js
module.exports = {
    publicPath: '/admin',
    /*
      Vue-cli3:
      Crashed when using Webpack `import()` #2463
      https://github.com/vuejs/vue-cli/issues/2463

     */
    /*
    pages: {
      index: {
        entry: 'src/main.js',
        chunks: ['chunk-vendors', 'chunk-common', 'index']
      }
    },
    */
    configureWebpack: {
        plugins: [
            // Ignore all locale files of moment.js
            new webpack.IgnorePlugin(/^\.\/locale$/, /moment$/),
            new webpack.DefinePlugin({
                "projectUrl": "\"" + require('./url') + "\"",
            }),
        ]
    },

    chainWebpack: (config) => {
        config.resolve.alias
            .set('@$', resolve('src'))
            .set('@api', resolve('src/api'))
            .set('@assets', resolve('src/assets'))
            .set('@comp', resolve('src/components'))
            .set('@views', resolve('src/views'))
            .set('@layout', resolve('src/layout'))
            .set('@static', resolve('src/static'))

        const svgRule = config.module.rule('svg')
        svgRule.uses.clear()
        svgRule
            .oneOf('inline')
            .resourceQuery(/inline/)
            .use('vue-svg-icon-loader')
            .loader('vue-svg-icon-loader')
            .end()
            .end()
            .oneOf('external')
            .use('file-loader')
            .loader('file-loader')
            .options({
                name: 'assets/[name].[hash:8].[ext]'
            })
        /* svgRule.oneOf('inline')
          .resourceQuery(/inline/)
          .use('vue-svg-loader')
          .loader('vue-svg-loader')
          .end()
          .end()
          .oneOf('external')
          .use('file-loader')
          .loader('file-loader')
          .options({
            name: 'assets/[name].[hash:8].[ext]'
          })
        */
    },

    css: {
        loaderOptions: {
            less: {
                modifyVars: {
                    /* less 变量覆盖，用于自定义 ant design 主题 */

                    /*
                    'primary-color': '#F5222D',
                    'link-color': '#F5222D',
                    'border-radius-base': '4px',
                    */
                },
                javascriptEnabled: true
            }
        }
    },

    devServer: {
        // 管理端开发口；API 在 :8000，热更新 sockjs 必须跟这里一致，不能指向 API。
        open: true,
        host: '0.0.0.0',
        port: 8081,
        disableHostCheck: true,
        sockHost: 'localhost',
        sockPort: 8081,
        proxy: {
            '/api': {
                // 同源 /api 转到本机 CIGC，保留 /api 前缀。Windows 浏览器不要直连 127.0.0.1:8000。
                target: process.env.ADMIN_PROXY || 'http://127.0.0.1:8000',
                ws: false,
                changeOrigin: true
            }
        }
    },

    lintOnSave: false,
    productionSourceMap: false,
    // babel-loader no-ignore node_modules/*
    transpileDependencies: []
}
