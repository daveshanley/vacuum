import { css } from 'lit';

export default css`
  .rule-icon {
    font-family: 'Arial';
    font-size: var(--sl-font-size-small);
    width: 20px;
    display: inline-block;
  }

  li {
    margin: 0;
    padding-left: 0;
  }

  li::after {
    content: '';
  }

  .details {
    margin-bottom: calc(var(--global-margin) / 2);
  }

  .details > .summary {
    background-color: var(--card-bgcolor);
    border: 1px solid var(--card-bordercolor);
    padding: 5px;
    border-radius: 3px;
  }

  .rule-violation-count {
    font-size: var(--sl-font-size-x-small);
    border: 1px solid var(--card-bordercolor);
    color: var(--tertiary-color);
    padding: 2px;
    border-radius: 2px;
  }

  .details.open .summary {
    background-color: var(--primary-color);
    color: var(--invert-font-color);
    font-weight: bold;
  }

  .details.open .rule-violation-count {
    background-color: var(--primary-color);
    color: var(--invert-font-color);
    border: 1px solid var(--invert-font-color);
    font-weight: normal;
  }

  .details.open .expand-state {
    color: var(--invert-font-color);
  }

  .details > div.violations {
    font-size: var(--sl-font-size-x-small);
    overflow-y: auto;
    overflow-x: hidden;
    border: 1px solid var(--card-bordercolor);
  }

  @media only screen and (max-width: 1200px) {
    .details > div.violations {
      max-height: 230px;
    }
  }

  .details > .summary::marker {
    color: var(--secondary-color);
  }

  .rule-description {
    font-size: var(--rule-font-size);
  }

  .summary:hover {
    cursor: pointer;
    background-color: var(--primary-color-lowalpha);
    color: var(--invert-font-color);
  }

  .summary:hover .expand-state {
    color: var(--invert-font-color);
  }

  .summary:hover .rule-violation-count {
    color: var(--invert-font-color);
    border: 1px solid var(--invert-font-color);
  }

  .violations {
    display: none;
    scrollbar-width: thin;
  }

  .violations::-webkit-scrollbar {
    width: 8px;
  }

  .violations::-webkit-scrollbar-track {
    background-color: var(--card-bgcolor);
  }

  .violations::-webkit-scrollbar-thumb {
    box-shadow: inset 0 0 6px rgba(0, 0, 0, 0.3);
    background: var(--primary-color-lowalpha);
  }

  .expand-state {
    color: var(--font-color);
    vertical-align: sub;
    height: 20px;
    width: 20px;
    display: inline-block;
  }

  .expand-state:hover {
    cursor: pointer;
    color: var(--primary-color);
  }

  .truncated {
    text-align: center;
    color: var(--error-color);
    border: 1px solid var(--error-color);
    padding: var(--global-padding);
    margin-bottom: 1px;
    margin-right: 1px;
  }
`;
