package task

import "testing"

func TestApplyTask_Run(t *testing.T) {
	at := &ApplyTask{
		Resources: []string{"resource.yaml"},
		Templates: map[string]string{
			"resource.yaml": "foo: bar",
		},
	}

	at.Run(Context{})
}

//func TestCreateTask_Run(t *testing.T) {
//	//k8sClient := fake.NewSimpleClientset()
//	//k8sClient.
//	type fields struct {
//		Resources []runtime.Object
//		Client    kubernetes.Clientset
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			c := &ApplyTask{
//				Resources: tt.fields.Resources,
//				Client:    tt.fields.Client,
//			}
//			if err := c.Run(); (err != nil) != tt.wantErr {
//				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
