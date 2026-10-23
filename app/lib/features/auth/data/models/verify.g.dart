// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'verify.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_VerifyRequest _$VerifyRequestFromJson(Map<String, dynamic> json) =>
    _VerifyRequest(
      code: json['code'] as String,
      email: json['email'] as String,
    );

Map<String, dynamic> _$VerifyRequestToJson(_VerifyRequest instance) =>
    <String, dynamic>{'code': instance.code, 'email': instance.email};

_VerifyResponse _$VerifyResponseFromJson(Map<String, dynamic> json) =>
    _VerifyResponse(
      data: TokenPair.fromJson(json['data'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$VerifyResponseToJson(_VerifyResponse instance) =>
    <String, dynamic>{'data': instance.data};
