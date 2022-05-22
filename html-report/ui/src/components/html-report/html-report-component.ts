import { BaseComponent } from '../../ts/base-component';
import { html } from 'lit';
import { CategoryActivatedEvent } from '../../model/events';

export class HtmlReportComponent extends BaseComponent {
  render() {
    return html`
      <div @categoryActivated=${this._categoryActivatedListener}>
        <slot name="navigation"></slot>
        <slot name="reports"></slot>
      </div>
    `;
  }

  _categoryActivatedListener(e: CustomEvent<CategoryActivatedEvent>) {
    const elements = document.querySelectorAll('category-report');
    const slot = this.shadowRoot
      .querySelector('slot')
      .assignedElements({ flatten: true });

    const description = slot[0]
      .querySelector('nav')
      .querySelector('#category-description');
    if (description) {
      description.innerHTML = e.detail.description;
    }

    elements.forEach((element: HTMLElement) => {
      if (element.id == e.detail.id) {
        element.style.display = 'block';
      } else {
        element.style.display = 'none';
      }
    });
  }
}
