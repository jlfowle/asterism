import { defineConfig } from 'eslint-define-config';
import reactPlugin from 'eslint-plugin-react';
import reactHooksPlugin from 'eslint-plugin-react-hooks';
import jsxA11yPlugin from 'eslint-plugin-jsx-a11y';

export default defineConfig([
  {
    files: ['**/*.js', '**/*.jsx'],
    languageOptions: {
      ecmaVersion: 2021,
      sourceType: 'module',
      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
      },
    },
    plugins: {
      react: reactPlugin,
      'react-hooks': reactHooksPlugin,
      'jsx-a11y': jsxA11yPlugin,
    },
    settings: {
      react: {
        version: 'detect',
      },
    },
    rules: {
      // Add any custom rules here
    },
  },
  {
    files: ['**/*.test.js', '**/*.test.jsx'],
    languageOptions: {
      globals: {
        jest: true,
      },
    },
  },
]);