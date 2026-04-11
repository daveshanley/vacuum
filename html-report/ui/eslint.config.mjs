import tseslint from 'typescript-eslint';

export default tseslint.config(
  {
    ignores: ['.DS_Store', 'build/', 'coverage/', 'node_modules/', 'webpack.config.js'],
  },
  ...tseslint.configs.recommended,
  {
    rules: {
      '@typescript-eslint/no-non-null-assertion': 'off',
      '@typescript-eslint/ban-ts-comment': 'off',
      '@typescript-eslint/no-unused-vars': 'error',
    },
  },
);
