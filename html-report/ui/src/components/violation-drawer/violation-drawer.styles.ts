import { css } from 'lit';
import { SyntaxCSS } from '../../model/syntax';

export default [
  SyntaxCSS,
  css`
    hr {
      border: 0;
      border-top: 1px dashed var(--secondary-color-lowalpha);
      margin-top: var(--global-margin);
      margin-bottom: var(--global-margin);
    }

    pre {
      overflow-x: auto;
    }

    pre::-webkit-scrollbar {
      height: 8px;
    }
    pre::-webkit-scrollbar-track {
      background-color: var(--card-bgcolor);
    }

    pre::-webkit-scrollbar-thumb {
      box-shadow: inset 0 0 6px rgba(0, 0, 0, 0.3);
      background: var(--primary-color-lowalpha);
    }

    p.violated {
      font-size: var(--sl-font-size-small);
    }

    pre {
      font-size: var(--sl-font-size-x-small);
    }

    a {
      font-size: var(--sl-font-size-small);
      color: var(--primary-color);
    }
    a:hover {
      background-color: var(--secondary-color);
      cursor: pointer;
      color: var(--invert-font-color);
    }
    h2 {
      margin-top: 0;
      line-height: 2.3rem;
      font-size: 1.4rem;
    }

    .backtick-element {
      background-color: black;
      color: var(--secondary-color);
      border: 1px solid var(--secondary-color-lowalpha);
      border-radius: 5px;
      padding: 2px;
    }

    section.select-violation {
      width: 100%;
      text-align: center;
    }
    section.select-violation p {
      color: var(--secondary-color-lowalpha);
      font-size: var(--sl-font-size-x-small);
    }

    section.how-to-fix p {
      font-size: var(--sl-font-size-x-small);
    }

    p.path {
      color: var(--secondary-color);
    }

    @media only screen and (max-width: 600px) {
      h2 {
        font-size: 1rem;
      }
    }
  `,
];
