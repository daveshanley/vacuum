import { BaseComponent } from '../../ts/base-component';
import { css, html } from 'lit';
import { property } from 'lit/decorators.js';
import { SyntaxCSS } from '../../model/syntax';

export class ViolationDrawerComponent extends BaseComponent {
  static get styles() {
    const listCss = css`
      ul {
        margin-top: 0;
      }
      .violation a {
        font-size: var(--sl-font-size-small);
        color: var(--font-color);
      }

      .violation a:hover {
        background-color: var(--secondary-color);
        cursor: pointer;
        color: var(--invert-font-color);
      }
      sl-drawer {
        --size: 80vh;
        backdrop-filter: blur(2px);
      }

      sl-drawer::part(panel) {
        background: var(--background-color-with-opacity);
        backdrop-filter: blur(3px);
      }
    `;
    return [SyntaxCSS, listCss];
  }

  @property()
  code: Element;

  @property()
  message: string;

  @property()
  path: string;

  @property()
  ruleId: string;

  render() {
    return html`
      <h2>${this.ruleId}</h2>

      ${this.message} ${this.code}
    `;
  }

  get drawer() {
    return this.shadowRoot.querySelector('sl-drawer');
  }

  public show() {
    //this.drawer.show();
  }

  public hide() {
    //this.drawer.hide();
  }
}
