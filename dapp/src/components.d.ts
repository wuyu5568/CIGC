import * as type from './components/components';
declare module '@vue/runtime-core' {
    export interface GlobalComponents {
        Container: typeof type.Container;
        BorderCard: typeof type.BorderCard;
        Toggle: typeof type.Toggle;
        Login: typeof type.Login;
    }
}