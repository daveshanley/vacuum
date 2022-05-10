import { html, css } from 'lit';
import { state } from 'lit/decorators.js';
import { BaseComponent } from '../../ts/base-component';

export interface Category {
  text: string;
  active: boolean;
}

export class RuleCategoryNavigationComponent extends BaseComponent {
  static get styles() {
    const rulesCss = css`
      .terminal-menu ul {
        flex-direction: row;
        place-items: center stretch;
        justify-content: center;
      }
    `;
    if (document.styleSheets.length > 0) {
      const { cssRules } = document.styleSheets[0];
      // @ts-ignore
      const globalStyle = css([
        Object.values(cssRules)
          .map(rule => rule.cssText)
          .join('\n'),
      ]);
      return [globalStyle, rulesCss];
    }
    return [rulesCss];
  }

  toggleCompleted(item: Category) {
    for (let x = 0; x < this._listItems.length; x++) {
      this._listItems[x].active = false;
    }
    item.active = !item.active;
    this.requestUpdate();
  }

  @state()
  private _listItems: Array<Category> = [
    { text: 'Category 1', active: true },
    { text: 'Category 2', active: false },
    { text: 'Category 3', active: false },
  ];

  render() {
    return html`
      <div class="terminal-nav">
        <nav class="terminal-menu">
          <ul>
            ${this._listItems.map(
              item => html` <li>
                <button
                  class=${item.active ? 'menu-item active' : 'menu-item'}
                  @click=${() => this.toggleCompleted(item)}
                >
                  ${item.text}
                </button>
              </li>`
            )}
          </ul>
        </nav>
      </div>
    `;
  }
}
