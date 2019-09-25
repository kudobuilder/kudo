package engine

/*

Package engine implements all the stuff for the engine.

To create a new task, the following steps need to be performed:

1. Create the task Go file and tests, implementing the Tasker interface. If your task has outputs, additionally implement the Outputter interface.
2. Embed the new task into the Task struct
3. Add a table test for the task to the UnmarshalJSON and Task tests
4. Add the instantiation of the new task and it's parameters into the UnmarshalJSON function. This allows Kubernetes to instantiate the task as the YAML is processed.
5. Add the new task to the Task.Run method so that it is correctly dispatched.









*/
