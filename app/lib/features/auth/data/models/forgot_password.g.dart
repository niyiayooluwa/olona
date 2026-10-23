// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'forgot_password.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_ForgotPasswordRequest _$ForgotPasswordRequestFromJson(
  Map<String, dynamic> json,
) => _ForgotPasswordRequest(email: json['email'] as String);

Map<String, dynamic> _$ForgotPasswordRequestToJson(
  _ForgotPasswordRequest instance,
) => <String, dynamic>{'email': instance.email};

_ResetPasswordRequest _$ResetPasswordRequestFromJson(
  Map<String, dynamic> json,
) => _ResetPasswordRequest(
  email: json['email'] as String,
  code: json['code'] as String,
  newPassword: json['new_password'] as String,
);

Map<String, dynamic> _$ResetPasswordRequestToJson(
  _ResetPasswordRequest instance,
) => <String, dynamic>{
  'email': instance.email,
  'code': instance.code,
  'new_password': instance.newPassword,
};
