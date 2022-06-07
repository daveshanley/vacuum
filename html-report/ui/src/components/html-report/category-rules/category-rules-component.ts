import { BaseComponent } from '../../../ts/base-component';
import { html } from 'lit';
import { RuleSelectedEvent } from '../../../model/events';
import { CategoryRuleComponent } from './category-rule-component';
import { customElement, property } from 'lit/decorators.js';
import categoryRulesStyles from './category-rules.styles';

@customElement('category-rules')
export class CategoryRulesComponent extends BaseComponent {
  static styles = categoryRulesStyles;

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
        <section class="no-violations">
          <p>All good in here, no rules broken!</p>
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
