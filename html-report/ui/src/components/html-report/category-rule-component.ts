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

      .rule-details::part(header) {
        margin: 0;
        padding: 3px;
      }

      .rule-details::part(summary) {
        font-family: Arial;
        padding: 2px;
      }
      .rule-details::part(base) {
        max-height: 500px;
        overflow-y: auto;
      }
      .rule-details::part(content) {
        margin: 0;
        padding: 0;
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

      <sl-details
          @click="${this._ruleClick}"
          summary="${this.ruleIcon} ${this.description} (${this.numResults})" 
          class="rule-details"> 
        <slot name="results">
      </sl-details>
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
