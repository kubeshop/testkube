// Package testworkflow holds the repository of Test Workflow executions and the filter of its lists.
//
// # Label scope
//
// Filter.LabelScope limits a list to the executions that an access rule allows. With a scope, a list
// keeps an execution when the workflow of the execution has no label with the key, or has the key with
// one of the values. A scope with no values keeps only the executions without the label. The scope
// joins the other conditions of the filter with AND, and it applies to the list, the summaries, the
// count, the totals and the finished executions, also on the fast path for one workflow name.
//
// The Control Plane sets the key to the Resource Group label, and the values to the groups that the
// caller may reach, by slug and by ID. A label selector cannot express this rule, because it joins a
// missing label and a label value with AND. The store escapes the dots of label keys, so the scope
// escapes its key in the same way.
package testworkflow
