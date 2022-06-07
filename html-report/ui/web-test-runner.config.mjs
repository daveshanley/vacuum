import { esbuildPlugin } from '@web/dev-server-esbuild';

export default {
    rootDir: '.',
    files: 'src/**/*.test.ts',
    concurrentBrowsers: 3,
    nodeResolve: true,
    plugins: [
        esbuildPlugin({
            ts: true,
            target: 'auto'
        })
    ],
    testRunnerHtml: testFramework => `
    <html lang="en-US">
      <head></head>
      <body>
        <script type="module" src="build/static/js/vacuumReport.js"></script>
        <script type="module" src="${testFramework}"></script>
      </body>
    </html>
  `,
};