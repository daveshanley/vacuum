// eslint-disable-next-line @typescript-eslint/no-var-requires
const path = require('path');

module.exports = {
  entry: {
    vacuumReport: './src/ts/vacuum-report.ts',
    hydrate: './src/ts/hydrate.ts'
  },
  devServer: {
    static: './build/static'
  },
  module: {
    rules: [
      {
        test: /\.tsx?$/,
        use: 'ts-loader',
        exclude: /node_modules/,
      },
      {
        test: /\.css$/,
        use: [
          'style-loader',
          'css-loader'
        ]
      },
    ],
  },
  resolve: {
    extensions: ['.tsx', '.ts', '.js'],
  },
  // optimization: {
  //   runtimeChunk: 'single'
  // },
  output: {
    path: path.resolve('build', 'static', 'js'),
    filename: '[name].js',
  }
};
