import {css} from 'lit';

export default css`
  ul {
    margin-top: 0;
  }

  .line {
    text-align: center;
    border-radius: 5px;
    min-width: 35px;
    max-width: 35px;
    background-color: var(--card-bgcolor);
    color: var(--tertiary-color);
    font-size: var(--sl-font-size-xx-small);
  }

  .violation {
    display: flex;

    border-top: 1px solid var(--card-bgcolor);
    border-bottom: 1px solid var(--card-bgcolor);
    font-size: var(--sl-font-size-x-small);
    color: var(--font-color);
  }

  .violation:hover {
    background-color: var(--secondary-color-x-lowalpha);
    cursor: pointer;
  }

  .violation.selected:hover {
    background-color: var(--secondary-color-lowalpha);
  }

  .code-render {
    display: none;
  }

  .message {
    margin-left: 5px;
  }

  .selected {
    background-color: var(--secondary-color-lowalpha);
  }

  .selected .line {
    color: var(--font-color);
  }

  .selected .message {
    font-weight: bold;
  }
`;
