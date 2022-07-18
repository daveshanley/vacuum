import {css} from 'lit';

export default css`
  ul.rule {
    margin: 0;
    padding: 0;
  }

  section {
    //max-height: 35vh;
    overflow-y: hidden;
  }

  p {
    font-size: var(--sl-font-size-small);
    margin: 0;
  }

  .symbol {
    font-family: Arial;
  }

  section.no-violations {
    border: 1px solid var(--terminal-green);
    padding: 20px;
    color: var(--terminal-green);
    text-align: center;
  }
`;
