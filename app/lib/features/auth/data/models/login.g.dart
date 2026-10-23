// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'login.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_LoginRequest _$LoginRequestFromJson(Map<String, dynamic> json) =>
    _LoginRequest(
      email: json['email'] as String,
      password: json['password'] as String,
    );

Map<String, dynamic> _$LoginRequestToJson(_LoginRequest instance) =>
    <String, dynamic>{'email': instance.email, 'password': instance.password};

_LoginResponse _$LoginResponseFromJson(Map<String, dynamic> json) =>
    _LoginResponse(
      data: TokenPair.fromJson(json['data'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$LoginResponseToJson(_LoginResponse instance) =>
    <String, dynamic>{'data': instance.data};
