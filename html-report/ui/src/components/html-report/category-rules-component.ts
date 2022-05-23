import { BaseComponent } from '../../ts/base-component';
import { BaseCSS } from '../../ts/base.css';
import { html, css } from 'lit';
import { RuleSelectedEvent } from '../../model/events';
import { CategoryRuleComponent } from './category-rule-component';

export class CategoryRulesComponent extends BaseComponent {
  static get styles() {
    const rulesCss = css`
      ul.rule {
        margin: 0;
      }
    `;

    return [BaseCSS, rulesCss];
  }

  render() {
    return html`
      <section @ruleSelected=${this._ruleSelected}>
        <ul class="rule">
          <slot></slot>
        </ul>
      </section>
    `;
  }

  private _ruleSelected(evt: CustomEvent<RuleSelectedEvent>) {
    const rules = this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });

    rules.forEach((catRule: CategoryRuleComponent) => {
      catRule.ruleSelected(evt.detail.id);
    });
  }
}
