import { BaseComponent } from '../../ts/base-component';
import { html, css } from 'lit';
import { RuleSelectedEvent } from '../../model/events';
import { CategoryRuleComponent } from './category-rule-component';
import { property } from 'lit/decorators.js';

export class CategoryRulesComponent extends BaseComponent {
  static get styles() {
    const rulesCss = css`
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
    `;

    return [rulesCss];
  }

  @property()
  id: string;

  @property()
  isEmpty: boolean;

  render() {
    if (!this.isEmpty) {
      return html`
        <section @ruleSelected=${this._ruleSelected}>
          <ul class="rule">
            <slot></slot>
          </ul>
        </section>
      `;
    } else {
      return html`
        <section>
          <p>
            <span class="symbol">✅</span> All good in here, no rules broken!
          </p>
        </section>
      `;
    }
  }

  get rules(): Element[] {
    const slots = this.shadowRoot.querySelector('slot');
    if (slots) {
      return slots.assignedElements({ flatten: true });
    }
  }

  private _ruleSelected(evt: CustomEvent<RuleSelectedEvent>) {
    this.rules.forEach((catRule: CategoryRuleComponent) => {
      if (catRule.ruleId != evt.detail.id) {
        catRule.otherRuleSelected();
      }
    });
  }
}
