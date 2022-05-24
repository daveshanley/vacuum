import { BaseComponent } from '../../ts/base-component';
import { html, css } from 'lit';
import { property } from 'lit/decorators.js';
import { RuleSelected, RuleSelectedEvent } from '../../model/events';

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

      details {
        margin-bottom: calc(var(--global-margin) / 2);
      }

      details > summary {
        background-color: var(--card-bgcolor);
        border: 1px solid var(--card-bordercolor);
        padding: 5px;
        border-radius: 3px;
      }

      details > div.violations {
        max-height: 300px;
        overflow-y: auto;
        border: 1px solid var(--card-bordercolor);
        padding: 5px;
      }

      details > summary::marker {
        color: var(--secondary-color);
      }

      details[open] summary span.rule-description {
        background-color: var(--primary-color);
        color: var(--invert-font-color);
      }

      summary span.rule-description:hover {
        cursor: pointer;
        background-color: var(--primary-color);
        color: var(--invert-font-color);
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

    return [iconCss];
  }

  @property()
  ruleId: string;

  @property()
  description: string;

  @property()
  numResults: number;

  @property()
  ruleIcon: number;

  ruleSelected(id: string) {
    [...this.shadowRoot.querySelectorAll('details')].map(details => {
      if (id != this.ruleId) {
        details.open = false;
      }
    });
    this.requestUpdate();
  }

  render() {
    return html`
      <details>
        <summary @click=${this._ruleSelected}>
          <span class="rule-icon">${this.ruleIcon}</span> 
          <span class="rule-description">${this.description}</span> (${this.numResults})
        </summary>
        <div class="violations">
          <slot name="results">  
        </div>
      </details>
    `;
  }

  private _ruleSelected() {
    const options = {
      detail: { id: this.ruleId },
      bubbles: true,
      composed: true,
    };
    this.dispatchEvent(
      new CustomEvent<RuleSelectedEvent>(RuleSelected, options)
    );
  }
}
