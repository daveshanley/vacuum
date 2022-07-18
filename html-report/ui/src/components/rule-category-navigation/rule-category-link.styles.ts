import { css } from 'lit';

export default css`
  li {
    padding-left: 0;
  }

  .active {
    background-color: var(--primary-color);
    color: var(--invert-font-color);
    font-weight: bold;
  }

  a {
    color: var(--primary-color);
    text-decoration: none;
  }

  a:hover {
    background-color: var(--primary-color);
    color: var(--invert-font-color);
  }

  @media only screen and (max-width: 600px) {
    a {
      font-size: 0.7rem;
    }
    li {
      padding-bottom: 0;
      margin-bottom: 0;
    }
  }
`;
