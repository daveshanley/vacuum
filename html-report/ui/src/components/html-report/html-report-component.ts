import {BaseComponent} from '../../ts/base-component';
import {css, html} from 'lit';
import {CategoryActivatedEvent} from '../../model/events';
import {CategoryRuleComponent} from './category-rules/category-rule-component';
import {ViolationDrawerComponent} from '../violation-drawer/violation-drawer-component';
import {CategoryRulesComponent} from './category-rules/category-rules-component';
import {CategoryReportComponent} from './category-report-component';
import {customElement} from 'lit/decorators.js';

@customElement('html-report')
export class HtmlReportComponent extends BaseComponent {
  static get styles() {
    const report = css`
      .html-report {
        height: 100%;
      }
    `;
    return [report];
  }

  render() {
    return html`
      <div
        class="html-report"
        @categoryActivated=${this._categoryActivatedListener}
        @violationSelected=${this._violationSelectedListener}
      >
        <slot name="navigation"></slot>
        <slot name="reports"></slot>
      </div>
    `;
  }

  _categoryActivatedListener(e: CustomEvent<CategoryActivatedEvent>) {
    const categoryReports = document.querySelectorAll('category-report');
    const categoryRules = document.querySelectorAll('category-rule');
    const categoryRuleGroup = document.querySelectorAll('category-rules');

    const violationDrawer = document.querySelector(
      'violation-drawer'
    ) as ViolationDrawerComponent;

    const slot = this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });

    const description = slot[0]
      .querySelector('nav')
      .querySelector('#category-description');

    if (description) {
      description.innerHTML = e.detail.description;
    }

    categoryReports.forEach((element: CategoryReportComponent) => {
      if (element.id == e.detail.id) {
        element.style.display = 'block';
      } else {
        element.style.display = 'none';
      }
    });

    // close every expanded rule back to closed.
    categoryRules.forEach((rule: CategoryRuleComponent) => {
      rule.otherRuleSelected();
    });

    categoryRuleGroup.forEach((rules: CategoryRulesComponent) => {
      if (rules.id == e.detail.id) {
        if (rules.rules && rules.rules.length <= 0) {
          rules.isEmpty = true;
        }
      }
    });

    if (violationDrawer) {
      violationDrawer.hide();
    }
  }

  _violationSelectedListener() {
    const violationDrawer = document.querySelector(
      'violation-drawer'
    ) as ViolationDrawerComponent;
    violationDrawer.show();
  }
}
