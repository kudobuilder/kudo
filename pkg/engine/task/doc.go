package task

/*

Package task contains all available tasks of the KUDO plan execution engine.

To create a new task, the following steps need to be performed:

1. Create the task Go file and tests, implementing the Tasker interface. New task has to implement the Run() method. Keep in mind that it supposed to be idempotent and will be called multiple time (on each controller reconciliation).
2. Introduce a new API level tasks spec and add it to the `TaskSpec` in operatorversion_types.go
3. Extend the Build() method in the task package to convert your task spec into a Tasker object

*/
