// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'error_response.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

_ApiErrorResponse _$ApiErrorResponseFromJson(Map<String, dynamic> json) =>
    _ApiErrorResponse(
      error: ApiErrorBody.fromJson(json['error'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$ApiErrorResponseToJson(_ApiErrorResponse instance) =>
    <String, dynamic>{'error': instance.error};

_ApiErrorBody _$ApiErrorBodyFromJson(Map<String, dynamic> json) =>
    _ApiErrorBody(
      code: json['code'] as String,
      message: json['message'] as String,
      details: (json['details'] as List<dynamic>?)
          ?.map((e) => FieldError.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$ApiErrorBodyToJson(_ApiErrorBody instance) =>
    <String, dynamic>{
      'code': instance.code,
      'message': instance.message,
      'details': instance.details,
    };

_FieldError _$FieldErrorFromJson(Map<String, dynamic> json) =>
    _FieldError(field: json['field'] as String, code: json['code'] as String);

Map<String, dynamic> _$FieldErrorToJson(_FieldError instance) =>
    <String, dynamic>{'field': instance.field, 'code': instance.code};
