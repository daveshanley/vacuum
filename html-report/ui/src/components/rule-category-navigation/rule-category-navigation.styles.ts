import {css} from 'lit';

export default css`
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    padding-left: 0;
  }

  @media only screen and (max-width: 600px) {
    ul {
      margin-block-start: 0;
      margin-block-end: 0;
      margin-inline-start: 0;
      margin-inline-end: 0;
      padding-inline-start: 0;
    }
  }
`;
