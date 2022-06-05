import { BaseComponent } from '../../ts/base-component';
import { css, html, TemplateResult } from 'lit';
import { property } from 'lit/decorators.js';
import { SyntaxCSS } from '../../model/syntax';

export class ViolationDrawerComponent extends BaseComponent {
  static get styles() {
    const listCss = css`
      hr {
        border: 0;
        border-top: 1px dashed var(--secondary-color-lowalpha);
        margin-top: var(--global-margin);
        margin-bottom: var(--global-margin);
      }

      pre {
        overflow-x: auto;
      }

      pre::-webkit-scrollbar {
        height: 8px;
      }
      pre::-webkit-scrollbar-track {
        background-color: var(--card-bgcolor);
      }

      pre::-webkit-scrollbar-thumb {
        box-shadow: inset 0 0 6px rgba(0, 0, 0, 0.3);
        background: var(--primary-color-lowalpha);
      }

      p.violated {
        font-size: var(--sl-font-size-x-small);
      }

      pre {
        font-size: var(--sl-font-size-x-small);
      }

      a {
        font-size: var(--sl-font-size-x-small);
        color: var(--primary-color);
      }
      a:hover {
        background-color: var(--secondary-color);
        cursor: pointer;
        color: var(--invert-font-color);
      }
      h2 {
        margin-top: 0;
        line-height: 2.3rem;
        font-size: 1.4rem;
      }

      .backtick-element {
        background-color: black;
        color: var(--secondary-color);
        border: 1px solid var(--secondary-color-lowalpha);
        border-radius: 5px;
        padding: 2px;
      }

      section.select-violation {
        width: 100%;
        text-align: center;
      }
      section.select-violation p {
        color: var(--secondary-color-lowalpha);
        font-size: var(--sl-font-size-x-small);
      }

      section.how-to-fix p {
        font-size: var(--sl-font-size-x-small);
      }

      p.path {
        color: var(--secondary-color);
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

  @property()
  howToFix: string;

  private _visible: boolean;

  render() {
    if (this._visible) {
      return html`
        <h2>${ViolationDrawerComponent.replaceTicks(this.message)}</h2>
        <p class="violated">
          Rule Violated:
          <a href="https://quobix.com/vacuum/rules/${this.ruleId}"
            >${this.ruleId}</a
          >
        </p>
        ${this.code}
        
        <h3>
          JSON Path
        </h3>
        <p class='path'>
          ${this.path}
        </p>
        
        </p>
        <hr/>
        <section class='how-to-fix'>
          <h3>How to fix violation</h3>
          <p>${this.howToFix}</p>
        </section>
        
      `;
    } else {
      return html`
        <section class="select-violation">
          <p>Please select a rule violation from a category.</p>
        </section>
      `;
    }
  }

  get drawer(): HTMLElement {
    return document.querySelector('violation-drawer') as HTMLElement;
  }

  public show() {
    this._visible = true;
    this.drawer.classList.add('drawer-active');
    this.requestUpdate();
  }

  public hide() {
    this._visible = false;
    this.drawer.classList.remove('drawer-active');
    this.requestUpdate();
  }

  private static replaceTicks(message: string): TemplateResult[] {
    const rx = /(`[^`]*`)/g;

    const sections = message.split(rx);
    const renders: Array<TemplateResult> = new Array<TemplateResult>();

    sections.forEach((section: string) => {
      if (section.match(rx)) {
        renders.push(html`
          <span class="backtick-element">${section.replace(/`/g, '')}</span>
        `);
      } else {
        if (section != '') {
          renders.push(html`${section}`);
        }
      }
    });
    return renders;
  }
}
