export const CategoryActivated = 'categoryActivated';
export const RuleSelected = 'ruleSelected';

export interface VacuumEvent {
  id: string;
}

export interface CategoryActivatedEvent extends VacuumEvent {
  description: string;
}

export type RuleSelectedEvent = VacuumEvent;
