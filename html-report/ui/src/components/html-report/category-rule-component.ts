import { BaseComponent } from '../../ts/base-component';
import { BaseCSS } from '../../ts/base.css';
import { html, css } from 'lit';
import { property } from 'lit/decorators.js';

export class CategoryRuleComponent extends BaseComponent {
  static get styles() {
    const iconCss = css`
      .rule-icon {
        font-family: Arial;
        font-size: var(--sl-font-size-small);
        with: 20px;
        display: inline-block;
      }

      li {
        margin: 0;
        padding-left: 0;
      }
      li::after {
        content: '';
      }
    `;

    return [BaseCSS, iconCss];
  }

  @property()
  ruleId: string;

  @property()
  description: string;

  @property()
  numResults: number;

  @property()
  ruleIcon: number;

  render() {
    return html`
      <li>
        <span class="rule-icon">${this.ruleIcon}</span>
        <span class="rule-description">
          <a @click="${this._ruleClick}">${this.description}</a>
          <sl-badge>${this.numResults}</sl-badge>
        </span>
      </li>
    `;
  }

  private _ruleClick() {
    const options = {
      detail: this.ruleId,
      bubbles: true,
      composed: true,
    };
    this.dispatchEvent(new CustomEvent('ruleSelected', options));
  }
}
